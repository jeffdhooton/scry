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
go build -trimpath -ldflags "-X main.version=scry-$sha" -o /tmp/scry.new ./cmd/scry
cp ~/go/bin/scry ~/go/bin/scry.pre-$sha
mv /tmp/scry.new ~/go/bin/scry
launchctl kickstart -k gui/$(id -u)/com.jhoot.scryd
scry version && scry doctor
```

Mini (it builds scry itself):

```sh
ssh mini 'cd ~/workspace/context-stack/scry && git pull --ff-only && \
  sha=$(git rev-parse --short HEAD) && \
  go build -trimpath -ldflags "-X main.version=scry-$sha" -o /tmp/scry.new ./cmd/scry && \
  cp ~/.local/bin/scry ~/.local/bin/scry.pre-$sha && mv /tmp/scry.new ~/.local/bin/scry && \
  launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd && sleep 2 && ~/.local/bin/scry version'
```

Plists: copy from `~/dotfiles/launchd/` to the LaunchAgents directory, then
`launchctl bootout gui/$(id -u)/<label>; launchctl bootstrap gui/$(id -u) <path>`.

## Check

```sh
scry doctor                      # Memory section: daemon, chain, ingest age, sweep age, queue
scry memory status --pretty      # counts, chain, queue depth, last ingest/sweep/extract
scry memory queue --pretty       # what is waiting, and why
tail -20 ~/.scry/logs/memory-sweep.log
ssh mini 'tail -50 ~/.scry/logs/scryd-launchd.log | grep "memory queue"'
```

## Roll back

Binary: `mv ~/go/bin/scry.pre-<sha> ~/go/bin/scry` (laptop) or
`mv ~/.local/bin/scry.pre-<sha> ~/.local/bin/scry` (mini), then kickstart the
daemon. The new key prefixes (`pq:`, `meta:`, `att:`) are ignored by older
binaries; the schema version is unchanged.

Store: `scry memory backup` writes `~/.scry/backups/memory-<utc>.badger` on the
store's machine. To restore: stop the daemon, `scry memory restore --from
<file>`, start it again.
