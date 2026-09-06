# Private identity/alias writer history — independent bounded review

2026-09-06. PASS for the pinned private, uncalled actual-writer primitive under its
explicit contract. No counterexample was found by the independent synthetic tests
or source inspection described here. This is not approval for production routing,
alias transfer, support selection, identity lifecycle changes, cleanup, deployment,
or completion of the active graph-quality objective.

I ran session orientation and read its output before other work. I read the complete
active objective, complete writer contract/source/supplied tests, full archived
identity-mutation design review, owner contract and implementation, journal source
and full archived corrected owner/journal reviews. I also inspected the generation
transaction guard, strict legacy entity decoder, key normalization/slug validation,
and non-test references to the new writer. No producer calls the primitive.

Baseline `42356e1a8ad6354c770c8fe53b3049951598f619` was independently exported with
git archive into `/tmp/scry-identity-writer-review.NTLeBc`. The three supplied pinned
additions were copied with apply_patch and their SHA-256 hashes matched exactly.
Only independent synthetic tests, this report, and test logs were added in that
private export. Candidate/shared files, live graph/replicas, provider/configuration,
remember/room/sweep/deploy operations and external writes were not touched. Every
database used by this review was a newly created synthetic temporary database.

## Independent disproof attempts

`writer_history_independent_test.go` contains six independent top-level tests:

1. Three deterministic generated runs execute 300 operations each across three
   actors and shared alias/entity keys. A separate expected-state model checks all
   900 generated contiguous sequence numbers, actors, exact keys, before/after
   presence and bytes, first images, final images and whole committed raw maps.
   Runs include puts, identical puts, delete/recreate, multi-actor alias replacement,
   repeated entity revisions, opaque old entity bytes and an existing empty alias
   value. Caller key/value buffers are overwritten after every call. Opaque unrelated
   attestation bytes survive and zero observer events are emitted. Display names
   deliberately diverge from entity slugs and remain accepted.

2. Eleven malformed new-entity cases reject body/key slug mismatch, empty name,
   zero/unrepresentable creation time, noncanonical whitespace, duplicate/unknown
   fields, invalid UTF-8, incomplete objects, arrays and trailing JSON. Each follows
   a successful staged write; its refusal is swallowed. The failed operation adds
   no successful entry, the finalizer is never reached, every original byte remains,
   and the primitive returns only its static sentinel.

3. Nine populated malformed/disallowed keys refuse deletion, including att:, fa:,
   ep:, ig:, empty/malformed entity keys and nonnormalized/invalid-UTF-8 aliases.
   Four additional existing alias values prove exact owner comparison: empty bytes,
   a NUL suffix, whitespace suffix and another actor all refuse deletion and roll
   back. The original journal's att: support is not inherited by this writer.

4. A capability created in one live admission scope is presented with a second
   genuine live owner's facade. It refuses and poisons the second scope, rolling
   back its prior staged write; the original independent scope remains usable.
   This changes the presented facade only, not internal history accounting fields.

5. Four stale-presence cases change a tracked key through the raw synthetic fault
   seam: delete a present key, or insert an existing empty value after a recorded
   deletion. Both the next write and freeze detect drift. Swallowed refusals roll
   back; failed freeze returns no partial snapshot.

6. A separate synthetic Badger competitor updates a key already read and staged
   by the writer. Freeze succeeds against its own transaction view, but actual
   commit returns ErrConflict. Only the competitor's expected bytes survive; no
   other writer key or event commits. The escaped conflicted writer is closed.

The unchanged five supplied top-level tests also pass. They additionally exercise
owned output arrays/maps, exact opaque first capture across recapture, actor/key/
value mismatches, nil/zero/root/ordinary/foreign capability refusals, absent deletion,
early/repeated freeze, late write, preexisting poisoning, closed scope, body and
finalizer panics, and real late oversized key/value and transaction-capacity errors.

Source inspection confirms capture happens before the actual Set/Delete; only a
successful call appends an entry; keys/values and output images are cloned; freeze
checks each tracked final image; and phase/owner errors poison the presented owned
facade. The primitive preserves static ErrTxnTooBig/ErrConflict classifications
while other details are replaced by its static error. An already recorded owner
error retains precedence at the outer boundary.

## Commands and results

From the private export with shell pipefail enabled:

```
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentWriter|TestWriterHistory' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

The targeted suite passed on its first run: eleven top-level tests, store 2.118s,
exit 0. Full exact output is `writer-targeted.log`. No failing tests or assertions
were removed or weakened, and no implementation changes were made.

The full noncached no-CGO suite also passed, exit 0, including all unchanged and
independent tests. Store completed in 37.282s, resolver in 15.699s, and daemon in
30.219s. Exact output is `writer-fullsuite.log`; no package result is cached.

## Limits

Actor attribution is mechanical and grants no alias-transfer or identity
authorization. Existing opaque before-images are not classified or adopted.
Identical puts are recorded operations; absent deletes refuse. The same private
field-integrity and nonconcurrent-use assumptions as the owner/fact ledger remain.
This report does not certify untracked changes to different keys, transient raw
writes restored between observations, or a complete baseline/final graph inventory.

The future cleanup design remains NO-GO until its fixed finalizer proves complete
relationship closure: every affected retained listing, index target and identity
state, including unchanged alias bytes whose target becomes absent. Exact baseline
defect preservation requires equality of those complete relationships. History alone
does not establish support, safe undo selection, recognized ownership, birth
disposition, production writer coverage, prevention, recall, live performance,
restored-replica behavior, real sweeps or any whole-goal clause.

## Exact pins

SHA-256; relative paths refer to this private export.

| Artifact | SHA-256 |
| --- | --- |
| IDENTITY_WRITER_CONTRACT.md | 04599cdf0902b8b0eacdcd411af6e94961a4226941dd930e6eec52228300996b |
| internal/memory/store/identity_writer_history.go | 25022d38ec62f80a4eb1ef1759938c2e651c0f8ed864e563ce40ad092eb2d148 |
| internal/memory/store/identity_writer_history_test.go | bc75a9fe46def626bb70195c48810de2a12930eea25757943bdbb8d73711d641 |
| internal/memory/store/writer_history_independent_test.go | 284e934cfc5d1b3c12dea46465529f63b735f52ce25f4796aa08a429fc180e04 |
| writer-targeted.log | 0d691d4e21d1c3a31b795d508ea025659de26c671b75b278fd3f6ba67eb9e3cc |
| writer-fullsuite.log | 22669a607b29ecb02b410495a1c8215eddfb2356ecd245a6b4d06cd1bb1c2a89 |
| docs/memory-repairs/identity-mutation-design-independent-review-2026-09-06.md | 94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3 |
| internal/memory/store/identity_admission_owner.go | 14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82 |
| internal/memory/store/identity_journal.go | d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80 |
| internal/memory/store/identity_legacy_anchor.go | 7a84b00d46977c2d8ce97c5b29f35763fa347bcd61f967d7178ef42a399a7de7 |
| /tmp/scry-admission-owner-sep06.t3WRXJ/OWNER_CONTRACT.md | 141ecaa7c5d1b91fd60bbd69eb1f1ae1077b33b148d561cd7f6941dd6758ce0f |

This report's hash is supplied separately to avoid a self-hash mismatch.
