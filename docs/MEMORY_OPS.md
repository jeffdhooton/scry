# Memory operations

How the shared memory store is deployed across the laptop and the Mac mini,
how writes get there, and how to deploy, check, and roll back. The audit
that motivated this layout is `docs/MEMORY_AUDIT_2026-09-02.md`.

## Topology

| Piece | Where | What |
|---|---|---|
| Store | mini, `/Users/jclaw/.scry/memory` | The one BadgerDB memory store |
| Daemon | mini, launchd `ai.jermes.scryd`, binary `~/.local/bin/scry` | Owns the store, runs the extraction queue worker |
| Extraction chain | mini, `~/.scry/config.yaml` `memory.models` | GLM-5.3-Flash on Z.ai, then DeepSeek. **The only place the chain is configured.** Keys come from `/Users/jclaw/.hermes/.env`, sourced by the launchd agent |
| Tunnel | laptop, launchd `com.jhoot.scry-memory-tunnel` | `ssh -L ~/.scry/shared-memory.sock:/Users/jclaw/.scry/scryd.sock mini`, 4096 descriptors |
| Laptop daemon | laptop, launchd `com.jhoot.scryd`, binary `/Users/jeff/go/bin/scry` | Code intelligence only. Its local memory store is unused and its extraction is deliberately dormant |
| Sweeps | laptop `com.jhoot.scry-memory-sweep`, mini `ai.jermes.scry-memory-sweep`, every 30 min | Distill new transcripts and queue them at the daemon. No API key needed |
| MCP hosts | laptop: Claude Code, Codex, OpenCode, Kimi | `scry-memory` server with `SCRY_MEMORY_SOCKET` pointing at the tunnel socket |

Plist sources live in `~/dotfiles/launchd/`; the installed copies are in
`~/Library/LaunchAgents` (laptop) and `/Users/jclaw/Library/LaunchAgents`
(mini).

## The write path

1. A client distills locally (`scry memory sweep`, `scry memory ingest`) or
   sends prose (`scry_remember`).
2. The daemon stores a `PendingEpisode` (`pq:` keys) and answers. A remember
   returns in milliseconds; its episode id is `sha256("manual:" + fact + day)`
   so a retried call lands on the same episode.
3. The queue worker (four goroutines) extracts each item with the model
   chain and applies it to the graph. Transport failures back off (30s, 1m,
   2m, 2m…) and retry forever. A reply no model can parse after three tries
   parks the item; it stays on disk and `scry memory queue retry <id>`
   replays it. A model that answers 401/402/403 is skipped for fifteen
   minutes.
4. Cursors advance only after the daemon accepted the episodes.

Timestamps the daemon keeps (`meta:` keys): last ingest (a transcript
episode queued), last sweep (a sweep reported), last successful extraction.
`scry memory status` shows them plus queue depth; `scry doctor` fails when
the last ingest is older than six hours.

## Sources swept

| Source | Root | Episode source |
|---|---|---|
| Claude Code | `~/.claude/projects/*/*.jsonl` | `claude-session` |
| Codex | `~/.codex/sessions/*/*/*/rollout-*.jsonl` | `codex-session` |
| Kimi Code | `~/.kimi-code/sessions/*/*/agents/*/wire.jsonl` | `kimi-session` |
| OpenCode | `~/.local/share/opencode/opencode.db` (SQLite, read via `sqlite3`) | `opencode-session` |
| loom | `~/.loom/runs/*` | `loom-run` |

## Deploy

Laptop (from the repo, on `main`):

```sh
sha=$(git rev-parse --short HEAD)
go build -trimpath -ldflags "-X main.Version=scry-$sha" -o /tmp/scry.new ./cmd/scry
cp ~/go/bin/scry ~/go/bin/scry.pre-$sha
mv /tmp/scry.new ~/go/bin/scry
launchctl kickstart -k gui/$(id -u)/com.jhoot.scryd
scry version && scry doctor
```

Mini (it builds scry itself):

```sh
ssh mini 'cd ~/workspace/context-stack/scry && git pull --ff-only && \
  sha=$(git rev-parse --short HEAD) && \
  go build -trimpath -ldflags "-X main.Version=scry-$sha" -o /tmp/scry.new ./cmd/scry && \
  cp ~/.local/bin/scry ~/.local/bin/scry.pre-$sha && mv /tmp/scry.new ~/.local/bin/scry && \
  launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd && sleep 2 && ~/.local/bin/scry version'
```

Plists: copy from `~/dotfiles/launchd/` to the LaunchAgents directory, then
`launchctl bootout gui/$(id -u)/<label>; launchctl bootstrap gui/$(id -u) <path>`.

## Measuring retrieval

Two question files, always reported together:

    scry memory bench --file docs/memory-bench/tuning-strict.json --top 20
    scry memory bench --file docs/memory-bench/tuning.json --top 20

The strict file pins each answer to one wording. The other allows a
question to list several phrasings of its answer under `any_of`, because
memory keeps more than one sentence for the same thing. The loose number
is always the higher one, which is exactly why the strict number is
reported beside it: six questions loosened once raised the score by six,
and reporting only that number would have measured the questions rather
than the retrieval. `--dir <store>` runs offline against a copy, which is
how ranking changes are tried without touching the live daemon.

## Check

```sh
scry doctor                      # Memory section: daemon, chain, ingest age, sweep age, queue
scry memory status --pretty      # counts, chain, queue depth, last ingest/sweep/extract
scry memory queue --pretty       # what is waiting, and why
tail -20 ~/.scry/logs/memory-sweep.log
ssh mini 'tail -50 ~/.scry/logs/scryd-launchd.log | grep "memory queue"'
```

### Shadow assessment operations

Assessment is off by default. Set `memory.assessment.mode: shadow` in the
**store daemon's** `~/.scry/config.yaml` and restart it. The model is pinned to
`jev-1.13.0`; the default target is approximately 20,000 input tokens and
the byte-bound counter deliberately underfills. Put `TYPESAFE_API_KEY` in the
daemon's launchd environment; exporting it from the laptop shell does not
configure a remote daemon. Assessment is observational and never enforces
memory admission. Switching back to `mode: off` and restarting stops dispatch
while retaining inspectable records.

For a bounded new-source trial, add a complete `trial` block under assessment:

```yaml
memory:
  assessment:
    mode: shadow
    concurrency: 1
    trial:
      starts_at: 2026-09-20T13:12:56Z
      expires_at: 2026-09-21T13:12:56Z
      max_requests: 100
```

These are the dates of the [September 20 trial](memory-assess/trial-2026-09-20/README.md),
not reusable relative defaults. All three trial fields are required. Source times
must be known and inside the window; old pending work and earlier raw/derived
history are excluded. Historical summary import is skipped. A future start waits
automatically. Expiry and the request limit suspend dispatch; `resume` cannot
bypass them. Change configuration and restart only for a deliberately extended
trial. The maximum is the sidecar's **lifetime dispatch reservation count**,
including failed and uncertain calls; changing dates, restarting or retention
does not reset it. The count commits synchronously before each request. Explicit
replays also consume a reservation and cannot send out-of-window evidence.

Status includes `store.dispatches` and an `error_counts` breakdown. Provider error
codes distinguish DNS, connection, TLS, transport, timeout/cancellation, response
read/size/JSON, usage, model, answer shape, and assertion-distribution failures.
HTTP status errors retain only their numeric code and cooldown. Unexpected
external errors remain `provider_request_failed`; no body, URL, credential or
arbitrary provider message is logged or persisted. Existing generic failures
cannot be classified retroactively. Status metrics scan at most 10,000 retained
jobs and explicitly report truncation.

Use `scry memory assess status --json` for effective configuration, boolean
key availability, blocked reason, coverage, queue counts, usage, estimated
cost and latency. `scry doctor` reports assessment health separately from
extraction health; off is healthy. Page through jobs with
`scry memory assess list --limit 50 --after CURSOR --json` and filter with
`--episode ID`. `scry memory assess show ID --json` omits retained evidence;
`--include-context` explicitly exposes the locally retained redacted packet
and sources. `scry memory assess replay ID` previews the packet;
`--live` requests a linked paid retry. Missing or expired evidence refuses
replay. After fixing a refusal or daemon credentials, use
`scry memory assess resume`; provider cooldown must have elapsed.

Retention limits are 30 days / 512 MiB for payloads, 90 days / 128 MiB for
metadata, and 10,000 jobs / 256 MiB for pending work. Check the status counts
and gaps before drawing conclusions from samples. Expiry removes logical
records; Badger disk space is reclaimed by value-log garbage collection, so
filesystem usage may lag. The assessment sidecar is separate from the memory
store backup and needs its own backup policy. Imported historical evidence is
a derived summary with source unavailable, not an original transcript.

## Throughput

Extraction is the slow part: GLM-5.3-Flash takes one to six minutes per
episode and cannot turn thinking off. The queue worker finds its own
concurrency — it starts at six in flight, halves when the provider answers
429, and widens by one when it is saturated and the provider has been
quiet for forty-five seconds, up to twenty-four. `scry memory status`
reports the current ceiling as `queue_limit`. A rate-limit refusal waits a
few jittered seconds and does not spend the item's attempt budget.

An episode the chain cannot finish after three escalating deadlines is
halved at a turn boundary and both halves are re-queued with fresh
budgets; only after three halvings is it parked. Manual remembers have
eight reserved slots and are dispatched first, so an agent never waits
behind a transcript backlog.

## Repair

`scry memory migrate` (dry run, then `--apply`) applies the current resolver
rules to the whole store under a backup and is a no-op when nothing is
left. `scry memory ingest --source kimi|opencode|claude|codex --path <p>
--force` re-queues a transcript the store already holds so its episodes
are re-applied (facts merge; entities refresh their repo refs and aliases).
`scry memory queue retry <id>` replays a parked item (all of them with no
id).

`scry memory repair-repos --apply` re-attaches repository refs: it walks
the same roots as the sweep, re-distills each transcript locally, and
tells the daemon which repository each episode ran in. No model is
called, so it costs nothing. Without `--apply` it reports what it found
and writes nothing; with it, it takes a store backup first. Run it on
each machine after a change to how repositories are attested, and once on
any machine whose sessions predate 2026-09-03.

`scry memory queue drop --match <text> --apply` removes queued work that
should never have been queued, a load test's synthetic lines being the
case it exists for. Without `--apply` it lists what it would remove.
Matching happens in the daemon, so it sees the whole queue rather than
the first fifty items `scry memory queue` prints. A queued transcript is
the only copy of that reading of a session; nothing else belongs here.

## Roll back

Binary: `mv ~/go/bin/scry.pre-<sha> ~/go/bin/scry` (laptop) or
`mv ~/.local/bin/scry.pre-<sha> ~/.local/bin/scry` (mini), then kickstart the
daemon. The new key prefixes (`pq:`, `meta:`, `att:`) are ignored by older
binaries; the schema version is unchanged.

Store: `scry memory backup` writes `~/.scry/backups/memory-<utc>.badger` on the
store's machine. To restore: stop the daemon, `scry memory restore --from
<file>`, start it again.
