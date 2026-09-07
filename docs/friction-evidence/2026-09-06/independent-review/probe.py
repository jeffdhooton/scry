"""Independent black-box acceptance and adversarial probes; temp service only."""
import concurrent.futures, copy, hashlib, json, os, pathlib, select, socket, subprocess

E = pathlib.Path(__file__).resolve().parent
CONFIG = json.loads((E / 'run-path.json').read_text())
T = pathlib.Path(CONFIG['temp'])
ROOT = pathlib.Path(CONFIG['root'])
HOME = T / 'home'
HOME.mkdir(exist_ok=True)
SOCKET = HOME / 'journal.sock'
ENV = {'HOME': str(HOME), 'SCRY_MEMORY_SOCKET': '/nonexistent/forbidden-memory'}
results = []
transcript = []
service = None

def check(name, condition):
    results.append({'check': name, 'pass': bool(condition)})
    assert condition, name

def start():
    global service
    service = subprocess.Popen([str(T/'service'), '-test.run=^TestFrictionServiceProcess$'],
        env=dict(ENV, SCRY_FRICTION_TEST_SERVICE='1', SCRY_FRICTION_TEST_HOME=str(HOME)),
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=open(T/'service.stderr', 'ab'))
    assert select.select([service.stdout], [], [], 10)[0], 'service readiness timeout'
    assert service.stdout.readline() == b'READY\n'

def stop(kill=False):
    if kill: service.kill()
    else: service.stdin.close()
    code = service.wait(timeout=10)
    assert kill or code == 0
    if SOCKET.exists(): SOCKET.unlink()

def cli(action, *args, event=None, ok=True):
    cmd = [str(T/'scry'), 'friction', '--socket', str(SOCKET), action, *args]
    p = subprocess.run(cmd, input=json.dumps(event) if event is not None else None,
        env=ENV, capture_output=True, text=True, timeout=10)
    transcript.append({'surface':'cli','args':[action,*args], 'code':p.returncode,
        'stdout':p.stdout,'stderr':p.stderr})
    assert (p.returncode == 0) == ok, p.stderr
    return json.loads(p.stdout) if ok else p.stderr

def rpc(action, params):
    with socket.socket(socket.AF_UNIX) as s:
        s.settimeout(10); s.connect(str(SOCKET))
        s.sendall((json.dumps({'jsonrpc':'2.0','id':1,'method':'friction.'+action,'params':params})+'\n').encode())
        response=json.loads(s.makefile('rb').readline())
    transcript.append({'surface':'rpc','action':action,'params':params,'response':response})
    return response

def mcp(action=None, params=None, profile='local'):
    req = {'jsonrpc':'2.0','id':1,'method':'tools/list'} if action is None else {
        'jsonrpc':'2.0','id':1,'method':'tools/call','params':{'name':'scry_friction_'+action,'arguments':params}}
    p=subprocess.run([str(T/'mcp-bridge'),str(SOCKET),profile],input=json.dumps(req)+'\n',
        env=ENV,capture_output=True,text=True,timeout=10)
    assert p.returncode == 0, p.stderr
    response=json.loads(p.stdout)
    transcript.append({'surface':'mcp','profile':profile,'request':req,'response':response})
    return response['result']

try:
    events=[json.loads(x) for x in (ROOT/'docs/workflow-pilots/2026-09-06-friction-pilot-01/friction-events.jsonl').read_text().splitlines()]
    check('Original fixture has exactly three observations',len(events)==3)
    start()
    receipts=[cli('record','-',event=e) for e in events]
    check('All pilot records created',all(r['created'] for r in receipts))
    stop(); start()
    for e,receipt in zip(events,receipts):
        check('Exact post-restart event '+e['event_id'],cli('get',e['event_id'])==e)
        retry=cli('record','-',event=e)
        check('Idempotent receipt '+e['event_id'],retry==dict(receipt,created=False))
    repo=events[0]['repository']
    page=cli('list','--repo',repo)
    check('CLI list exact three and no cursor',page=={'events':events})
    initial=cli('review','--repo',repo)
    check('Initial review no inferred recurrence or costs',all(g['distinct_runs']==1 and not g['recurring'] and g['measured_user_time_cost_seconds'] is None for g in initial['groups']))
    second=copy.deepcopy(events[1]); second.update(event_id='INDEPENDENT-second',run_id='INDEPENDENT-synthetic-run-2',observed='Synthetic verifier recurrence, not a real incident.')
    check('Second run record acknowledged',rpc('record',second)['result']['created'])
    stop(kill=True); start()
    check('Acknowledged event survives process kill',rpc('get',{'event_id':second['event_id']})['result']==second)
    report=cli('review','--repo',repo)
    g=next(g for g in report['groups'] if g['signature']==second['signature'])
    check('Exactly two distinct runs',g['distinct_runs']==2 and set(g['run_ids'])=={events[1]['run_id'],second['run_id']} and g['recurring'])
    check('Original unresolved outcome and full evidence preserved',next(e for e in g['events'] if e['event_id']==events[1]['event_id'])==events[1])
    check('Proposals cited and recommendations only',report['recommendations_only'] and {p['event_id']:p['change'] for p in g['proposals']}=={e['event_id']:e['proposed_change'] for e in [events[1],second]})
    (E/'acceptance-review.json').write_text(json.dumps(report,indent=2)+'\n')
    for profile,want in [('all',4),('local',4),('memory',0)]:
        check('MCP profile '+profile,sum(t['name'].startswith('scry_friction_') for t in mcp(profile=profile)['tools'])==want)
    check('Memory profile refuses journal call',mcp('get',{'event_id':events[0]['event_id']},'memory')['isError'])
    check('MCP exact get',json.loads(mcp('get',{'event_id':events[1]['event_id']})['content'][0]['text'])==events[1])
    check('MCP retry record',json.loads(mcp('record',events[0])['content'][0]['text'])==dict(receipts[0],created=False))
    check('MCP list exact',json.loads(mcp('list',{'repository':repo,'run_id':events[0]['run_id']})['content'][0]['text'])=={'events':events})
    check('MCP review exact',json.loads(mcp('review',{'repository':repo})['content'][0]['text'])==report)
    altered=dict(events[0],resolution='Falsely fixed')
    check('Conflict rejects overwrite',rpc('record',altered)['error']['code']==-32009)
    check('Missing returns -32004',rpc('get',{'event_id':'absent'})['error']['code']==-32004)
    for name,event in [('authorization',dict(events[0],change_approved=True)),('unknown',dict(events[0],typo='bad')),('oversize',dict(events[0],observed='x'*17000)),('path',dict(events[0],repository='/a/../b'))]:
        check('RPC rejects '+name,rpc('record',event)['error']['code']==-32602)
    check('MCP rejects unknown fields',mcp('record',dict(events[0],typo='bad'))['isError'])
    check('CLI rejects repo rewrite','omit --repo' in cli('record','-','--repo','/other',event=events[0],ok=False))
    # Distinct clients exercise the production daemon concurrency boundary.
    parallel=dict(events[0],event_id='parallel',run_id=second['run_id'],repository='/independent/concurrency',distinct_prior_runs_verified=999,occurrences_observed_this_run=999)
    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
        answers=list(pool.map(lambda _:rpc('record',parallel),range(32)))
    check('32 concurrent retries insert exactly once',sum(a['result']['created'] for a in answers)==1 and len({a['result']['sha256'] for a in answers})==1)
    parallel2=dict(parallel,event_id='parallel2'); rpc('record',parallel2)
    one=rpc('review',{'repository':parallel['repository']})['result']['groups'][0]
    check('Same-run agents and asserted counts never create recurrence',one['distinct_runs']==1 and not one['recurring'])
    # Capacity and lexical cursor checks over 101 valid records.
    for i in range(101):
        e=dict(events[0],event_id=f'bounded-{i:03d}',repository='/independent/bounds')
        assert rpc('record',e)['result']['created']
    pages=[]; after=''
    while True:
        p=rpc('list',{'repository':'/independent/bounds','limit':13,'after':after})['result']
        pages+=p['events']; after=p.get('next_after','')
        if not after: break
    check('Pagination returns 101 distinct ordered IDs',len(pages)==101 and [e['event_id'] for e in pages]==sorted({e['event_id'] for e in pages}))
    check('Review refuses 101 records',rpc('review',{'repository':'/independent/bounds'})['error']['code']==-32602)
    check('List refuses limit 101',rpc('list',{'repository':repo,'limit':101})['error']['code']==-32602)
    check('Review refuses cursor',rpc('review',{'repository':repo,'after':events[0]['event_id']})['error']['code']==-32602)
    check('Until exclusive',rpc('list',{'repository':repo,'until':events[0]['recorded_at']})['result']['events']==[])
    check('Since inclusive',len(rpc('list',{'repository':repo,'since':events[0]['recorded_at']})['result']['events'])==4)
    # Extreme finite costs must not produce an unserializable review.
    for i in range(2):
        e=dict(events[0],event_id=f'overflow-{i}',repository='/independent/overflow',measured_user_time_cost_seconds=1e308)
        check('Finite extreme cost accepted '+str(i),rpc('record',e)['result']['created'])
    overflow=rpc('review',{'repository':'/independent/overflow'})
    (E/'overflow-response.json').write_text(json.dumps(overflow,indent=2)+'\n')
    check('Finite cost overflow explicitly refuses complete review',overflow.get('error',{}).get('code')==-32602 and 'overflows' in overflow['error']['message'])
    check('Overflow does not lose either stored event',len(rpc('list',{'repository':'/independent/overflow'})['result']['events'])==2)
    check('Overflow can be scoped to a single event',rpc('get',{'event_id':'overflow-0'})['result']['measured_user_time_cost_seconds']==1e308)
    check('No memory store created',not (HOME/'memory').exists())
    check('Original pilot still exactly preserved after probes',all(rpc('get',{'event_id':e['event_id']})['result']==e for e in events))
finally:
    if service and service.poll() is None: stop()
    (E/'results.json').write_text(json.dumps(results,indent=2)+'\n')
    (E/'transcript.json').write_text(json.dumps(transcript,indent=2)+'\n')
    print(json.dumps({'checks':len(results),'passed':sum(r['pass'] for r in results),'failures':[r for r in results if not r['pass']]},indent=2))
