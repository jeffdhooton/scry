# Background change review

Scry can prepare an independent review while a coding agent works. One read-only
model call returns a change explanation, regression findings, and test gaps.
Every record retains the captured evidence, code fingerprint, context hash,
model, timestamps, duration, and available token usage. Findings are provisional
and never enter the memory graph automatically.

## Configure

Add a separate `review` section to `~/.scry/config.yaml`; preserve `memory`:

```yaml
review:
  enabled: true
  repos:
    - /absolute/path/to/repository
  include_memory: true
  protocol: anthropic
  base_url: https://api.z.ai/api/anthropic
  model: glm-5.3-flash
  api_key_env: Z_AI_API_KEY
  quiet_seconds: 30
  poll_seconds: 5
  timeout_seconds: 120
  max_input_bytes: 48000
  max_output_tokens: 4096
  max_requests_per_day: 10
  retain: 100
  exclude:
    - private/**
```

The provider must speak Anthropic Messages (`anthropic`) or OpenAI chat
completions (`openai`). All routing and credential variable names are explicit.
Keys come from the daemon launcher environment, never YAML. The laptop's
existing launcher sources `~/.secrets.zsh`. Restart that existing service after
configuration changes. Review runs on the local code daemon; `memory.socket`
only controls where supporting decisions are retrieved. `include_memory` opts
into including recalled facts in the configured review provider's input.

The service defaults off. Only allowlisted repositories are captured. Polling
waits for an unchanged fingerprint across the quiet period; it does not depend
on successful indexing. One model request runs at a time across repositories.
A snapshot/context/model combination is attempted once, including failures.
Restarting does not reset attempts or daily reservations. Limits use UTC days.
Failed and interrupted calls consume allowance; there are no automatic HTTP
retries. GLM reasoning consumes the output allowance, so a truncated response
is a visible failed review, never a clean result.

Optionally set `max_daily_usd`, `input_usd_per_million`, and
`output_usd_per_million` together using your provider's applicable rates. The
service reserves a conservative input/output upper bound before each request;
it does not refund unused reservations. Recorded cost is an estimate computed
from actual returned tokens and configured rates, not a billing receipt. Without
rates, costs remain unknown and request/token bounds still apply. Failed replies
retain usage when the provider supplied it.

## Use from any agent

```sh
scry review status --pretty
scry review preview --repo /absolute/repo --pretty
scry review run --repo /absolute/repo
scry review list --repo /absolute/repo --pretty
scry review get RECORD_ID --pretty
```

`preview` performs local capture and read-only context retrieval, with no model
call. `run` queues a request at the next stable checkpoint; it returns promptly.
`list` returns the latest 20 record summaries; `get` includes full evidence.
Results remain inspectable after restart. Recent records are retained under
`~/.scry/reviews/state.json`; the separate usage/deduplication fields are not
removed when old results expire. Do not delete that journal to reset limits.

MCP equivalents are `scry_review_status`, `scry_review_preview`,
`scry_review_run`, `scry_review_list`, and `scry_review_get`, in the local/all
profiles only. Restart an existing MCP host connection to discover new tools.
Foreground agents should read available reviews at a checkpoint and assess
findings against their cited evidence. Scry does not interrupt the foreground
agent or block a commit. Different clients retrieve the same durable records.

## Freshness and evidence

A fingerprint includes HEAD, staged blobs/modes, working files/modes, deletions,
and nonignored untracked files. Staged content is labeled separately when it
differs from the working tree. A before/after fingerprint guards evidence
assembly. Retrieval recaptures evidence and compares both code and context:
`current`, `stale`, or `unknown` when capture is unavailable. A change anywhere
in the repository conservatively invalidates a result; a change to recalled
supporting decisions also invalidates it. Old findings remain historical.

The evidence package contains bounded before/after source, indexed references,
current caller source, and recalled facts with episode pointers when available.
Index results are explicitly retrieval hints, not proof of complete caller
coverage. Missing indexes, unavailable memory, truncation and omissions are
reported to both the model and clients. No model tools or repository execution
are available. The collector does not execute Git filters or external diffs.
Symlink targets are not followed. Secret-like paths and configured exclusions
are omitted; this path filter cannot discover arbitrary secrets in ordinary
source files. Unmerged indexes and submodules fail capture explicitly.

## Acceptance and ongoing evaluation

`internal/daemon/review_acceptance_test.go` contains the permission-change
acceptance scenario: three callers, an emergency-access exception supported by
an attributed prior decision, and a regression removing that exception. The
normal test uses a controlled reviewer to validate the pipeline, stable
checkpoint, shared result retrieval, code/memory staleness, timing/usage, and
absence of memory writes. It does not establish model quality.

An opt-in live variant runs the same fixture with a configured real provider.
Its output is retained for inspection; a useful result identifies the removed
exception and cites both code and memory evidence. Live inference costs money
and is not part of the regular test suite.

For real coding sessions, record whether each finding was useful, a false
positive, or inconclusive; compare its completion timestamp with the foreground
agent's finish time, record review-reading time, and compare token/cost usage.
A successful fixture is initial validation, not a measured production recall or
false-positive rate. No automatic notification is sent, so interruption cost is
limited to the checkpoints where agents/users choose to inspect findings.

Provider responses 401, 402, 403, and 429 persist a block across restarts. Fix the
provider availability issue, inspect status, then explicitly run
`scry review resume`. Resume clears the provider block only; it does not reset
usage counters or reattempt the same failed snapshot. It is intentionally a
CLI-only operator action, not an MCP tool.

To repeat the model evaluation with an available provider (one paid request):

```sh
SCRY_REVIEW_LIVE=1 \
SCRY_REVIEW_LIVE_PROTOCOL=anthropic \
SCRY_REVIEW_LIVE_BASE_URL=https://api.z.ai/api/anthropic \
SCRY_REVIEW_LIVE_MODEL=glm-5.3-flash \
SCRY_REVIEW_LIVE_API_KEY_ENV=Z_AI_API_KEY \
SCRY_REVIEW_LIVE_REPORT=/absolute/path/regression.json \
go test ./internal/daemon -run '^TestReviewPermissionLiveAcceptance$' -count=1 -v
```

The named credential must already be in the process environment. To evaluate
false positives, add `SCRY_REVIEW_LIVE_CASE=control` and choose a distinct report
path. This preserves the emergency exception in an equivalent refactor. Do not
run the control while the provider is blocked. Fixture calls use their own
one-request temporary ledger; account for them alongside daemon usage.
