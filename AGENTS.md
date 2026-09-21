# Agent instructions

## Orient before setup

Run `scry memory orient --cwd .` at session start. Recall unfamiliar services,
machines and prior decisions before guessing; use `scry_recall` and, for their
relationships, `scry_memory_path`. Store durable setup decisions with
`scry_remember`, without credentials.

For installation or troubleshooting, read the [README setup guide](README.md#setup)
and [memory operations](docs/MEMORY_OPS.md). First identify the requested domains,
the installed binary/version, and which machine owns the memory store. Preserve
existing model chains, service launchers, credential locations and trial limits.
The release installer may lag this checkout; confirm that the installed binary
supports the requested commands, including `scry memory assess status --help`
when configuring Jev.

## Dependencies and model contract

- **Code/git/schema/HTTP:** no AI model or provider key required. Install the
  relevant language toolchain/indexer and project dependencies; git needs `git`,
  schema needs a reachable database DSN. See the README prerequisite table.
  Building this checkout requires the Go version declared in `go.mod` (1.26.2).
- **Memory extraction:** requires a generative model with a Messages-compatible
  API, a valid provider account/key with model access and credit, and outbound
  HTTPS from the store daemon. Scry does not bundle that model. The code defaults
  to DeepSeek `deepseek-v4-flash` at `https://api.deepseek.com/anthropic`.
- **Jev assessment:** separately optional, off by default. Requires extracted
  candidate facts, TypeSafe access to `jev-1.13.0`, and `TYPESAFE_API_KEY` in the
  store daemon. It sends redacted evidence to
  `https://api.typesafe.ai/v1/systemone` and stores observational assessments in
  `~/.scry/memory-assess/`. It does not replace the extraction model or enforce
  memory admission or recall ranking.
- **Storage/ingestion:** writable `~/.scry/`, readable supported transcripts,
  and a separately configured sweep scheduler for unattended ingestion. Badger
  and local retrieval are embedded; no external database/vector service or GPU
  is needed. Existing memories remain queryable without model keys.

Extraction key lookup is exact:

1. With no `memory.models` entries in `~/.scry/config.yaml`, use
   `SCRY_MEMORY_API_KEY`, then `DEEPSEEK_API_KEY` if empty. Optional
   `SCRY_MEMORY_MODEL` / `SCRY_MEMORY_BASE_URL` override defaults; an explicit
   base URL requires an explicit model. An OpenAI-style endpoint alone is not
   sufficient: it must implement the Messages API.
2. A nonempty `memory.models` chain replaces the environment model/base URL
   overrides. Every entry needs `model`; omitted `base_url` means DeepSeek.
   Explicit `api_key_env` reads only that variable. Without it, the default key
   lookup applies. Missing fallback credentials invalidate the chain too.
3. `ANTHROPIC_API_KEY` is never implicitly consulted. Using Anthropic requires
   explicitly selecting its endpoint/model and key variable. `TYPESAFE_API_KEY`
   is for Jev only and cannot activate extraction.

## Credentials, services and shared memory

Scry reads the process environment; it does **not** automatically load `.env`,
shell profiles or a password manager for API credentials. YAML stores model IDs,
endpoints and key **variable names**, never secret values. Do not print keys,
copy them into repository files, or use raw environment dumps for diagnostics.

For a new installation, the README suggests a private mode-600 file at
`~/.config/scry/credentials.env`. This is a launcher convention, not a Scry
discovery path. Load it explicitly before launching, for example:

```sh
set -a
. "$HOME/.config/scry/credentials.env"
set +a
scry start --foreground
```

The file needs the extraction key (or every configured `api_key_env`) and,
only when using Jev, `TYPESAFE_API_KEY`. See the README for placeholder contents
and launchd/systemd examples. Use absolute binary/file paths in supervisors
and make indexing toolchains available on the daemon's PATH. Update the actual
service launcher and restart the store daemon after key/config changes;
starting a second daemon command does not update an existing process.

For this workspace's existing shared deployment, the Mini owns memory and its
launcher sources `/Users/jclaw/.hermes/.env`. Put provider keys there using the
existing private editing workflow. Laptop exports do not configure the Mini.
See `docs/MEMORY_OPS.md` for service labels, binaries, tunnel and rollback paths.

Memory routing uses `SCRY_MEMORY_SOCKET`, then `memory.socket` in config, then
the local socket. The client machine needs a working tunnel/socket; the store
machine needs the model configuration and credentials. Configuring a remote
memory socket suppresses local daemon extraction/assessment. Check both ends.

`scry setup` installs Claude Code integration, including the embedded
[`internal/setup/SKILL.md`](internal/setup/SKILL.md). It does not provision model
accounts/keys, install a daemon service, establish tunnels or schedule sweeps.
Custom installed skills are preserved unless replacement is explicitly forced.
Other MCP hosts need their own configuration with an absolute binary path.

## Verify the requested setup

Start with status and previews to inspect configuration without submitting new
ingestion or assessment work:

```sh
scry doctor --json
scry status
scry memory status
scry memory sweep --dry-run
scry memory assess status --json
scry memory assess list --limit 20 --json
```

Interpret diagnostics for the requested domains: intentionally dormant memory
does not invalidate a code-only installation. For memory, verify the intended
`models`, `dormant: false`, `worker_running: true`, ingestion/tunnel health and
successful extraction of an authorized episode. For Jev, verify effective
`configuration.mode`, `key_available`, limits, blocked reason and an inspectable
completed assessment. Key availability only means nonempty; it does not prove
provider acceptance. Report unresolved typed failures instead of claiming success.

Follow the [bounded shadow-trial configuration](docs/MEMORY_OPS.md#shadow-assessment-operations).
All three trial fields (`starts_at`, `expires_at`, `max_requests`) are required.
Use deliberately chosen dates; historical trial dates are not reusable defaults.
The cap counts lifetime dispatch reservations, including failed/uncertain calls;
restart and retention do not reset it. Expiry/cap stops cannot be bypassed by
`resume`. Fix credentials/access/cooldown before resuming other blocked states.
To disable assessment, set mode `off` and restart; retain the sidecar records.

Honor existing user authorization. Complete requested memory/Jev setup within
that scope without asking again, but code indexing or documentation work alone
does not authorize paid ingestion, live evaluations, historical backfill, or
extending a trial. `memory backfill` and assessment fixture `--live` run provider
calls in the calling process and need its credentials; a successful fixture run
does not verify the daemon's environment. Never use a paid run just to validate
a documentation edit.

When changing setup behavior, keep README, this file and the embedded setup
skill consistent. Check defaults against `internal/memory/extract/provider.go`,
`internal/memory/extract/resolve.go`, `internal/config/assessment.go`, and
`internal/daemon/memory_assessment.go`.

## Project skills

Use the [TypeSafe skill](.agents/skills/typesafe-ai/SKILL.md) when working on
TypeSafe/Jev features in this project. Read the skill and follow its live
documentation workflow before designing or changing the integration.
