# Independent friction journal verification

**Verdict: PASS against the original user-approved acceptance bar.** No unresolved
blocking defect found in the final reviewed source. This report covers local
source delivery, not installation or production rollout.

The reviewer was a fresh agent, separately authorized by the user. It reviewed
the implementation and wrote its own black-box assertions. The builder-authored
tests were not the sole basis for this verdict. Reviewed baseline: `64c0cde`.
Exact final SHA-256 values for implementation, tests, documentation, pilot fixture
and protected user documents are in [reviewed-hashes.json](reviewed-hashes.json).
Built binary hashes are in [binary-hashes.json](binary-hashes.json).

## Original acceptance

All of these independently passed against the actual CLI and production daemon
handlers, using only an isolated `/tmp/sfiv-*` home/socket:

- Loaded the three original JSONL pilot observations through `friction record`.
- Stopped the isolated service, restarted it, and retrieved each complete event
  through `friction get`, comparing parsed objects with the original fixture.
  Historical `workaround_only` and `pilot_in_progress` outcomes remained exact.
- Retried all three records: each returned the original receipt hash with
  `created:false`; list contained exactly the original three events.
- Recorded a clearly synthetic second run with the policy-conflict signature,
  killed the service process after its acknowledgment, restarted it, and retrieved
  that event exactly.
- `friction review` returned four events, with exactly two distinct run IDs in
  the recurring signature. Original full observations, evidence references/hashes,
  and authored proposed corrections were retained and cited by event ID.
- No memory store or extraction service was created. Provider credentials and
  executables were absent from the subprocess environment. The service fixture
  starts only production friction handlers; there is no model/extraction path.
- Reviews marked `recommendations_only:true`; attempted `change_approved:true`
  was rejected. Source inspection found no proposal activation, instruction
  writing, automatic capture, scheduling, or memory admission changes.

The exact accepted review is [acceptance-review.json](acceptance-review.json).
The synthetic second run demonstrates counting behavior only; it is not evidence
of a second real task or of time savings.

## Additional independent checks

[results.json](results.json) records **46/46 checks passing**.
[transcript.json](transcript.json) retains requests/responses and CLI outputs.

- All four MCP tools exercised the real production MCP server and real daemon
  handler over the explicit temporary socket: record, get, list, review.
  `all` and `local` exposed four tools; `memory` exposed none and refused a call.
- Thirty-two concurrent RPC clients retrying one ID created exactly one record
  and returned one receipt hash. Another event from the same run did not create
  recurrence, even with authored occurrence/prior-run assertions set to 999.
- Conflicting overwrite returned `-32009`; missing get returned `-32004`.
  Unknown fields, oversized events, unclean repository paths, approval, oversized
  list limits, and review cursors were rejected explicitly. CLI `--repo` could
  not rewrite record attribution. MCP forwarded unknown fields for rejection.
- Pagination enumerated 101 unique IDs in lexical order, in pages of 13. Review
  refused the 101-event scope rather than truncating. `since` was inclusive and
  `until` exclusive at the original timestamp.
- A separate independently authored store probe cancelled a listing, installed
  an unsupported schema marker in a temporary database, attempted reopening,
  and verified the exact marker and event bytes were preserved after rejection.
  See [store-probe.log](store-probe.log).
- [scope-preservation.json](scope-preservation.json) records byte comparisons
  against `64c0cde`: 246 graph/memory-related baseline files were unchanged.
  The existing graph registrations remain unchanged in the shared daemon/MCP
  files; their diffs only add friction registration/dispatch/lifecycle state.
  Both preexisting untracked user documents retain their original hashes.

## Defect found and corrected during review

The pre-fix build accepted two individually finite `1e308` costs, then returned
`-32603` with `marshal result: json: unsupported value: +Inf` when reviewing their
sum. This was a real extreme-input aggregation defect, outside the original pilot
data. It was reported to the parent; the parent added an explicit overflow guard
and regression test. The reviewer did not edit implementation or builder tests.

The final rebuilt service returns `-32602` with an actionable overflow error,
without a partial report or loss of either stored event. The entire independent
acceptance/probe suite was rerun against the corrected build, passing 46 checks.
Before evidence is retained in [before-results.json](before-results.json),
[before-overflow-response.json](before-overflow-response.json),
[before-transcript.json](before-transcript.json),
[before-reviewed-hashes.json](before-reviewed-hashes.json), and
[binary-hashes-before.json](binary-hashes-before.json).
Final overflow evidence: [overflow-response.json](overflow-response.json).

## Commands and reproducibility

All Go builds/runs used `CGO_ENABLED=0`. From the repository root, the actual
commands were:

```sh
CGO_ENABLED=0 go build -o /tmp/sfiv-laajqs9d/scry ./cmd/scry
CGO_ENABLED=0 go test -c -o /tmp/sfiv-laajqs9d/service ./internal/daemon
CGO_ENABLED=0 go build -o /tmp/sfiv-laajqs9d/mcp-bridge ./docs/friction-evidence/2026-09-06/independent-review/mcp_bridge.go
python3 docs/friction-evidence/2026-09-06/independent-review/probe.py
CGO_ENABLED=0 go run ./docs/friction-evidence/2026-09-06/independent-review/store_probe.go /tmp/sfiv-laajqs9d
```

All commands exited 0. The pre-fix Python probe deliberately logged the overflow
as a failed check rather than aborting; its summary was 43/44. Final summary is
[probe.log](probe.log): 46/46. For reproduction, allocate a fresh short temp path
and update [run-path.json](run-path.json). The probe expects fresh storage.

The two independent Go adapters have been archived with `.go.txt` suffixes after
execution so they do not add packages to `go test ./...`:
[mcp_bridge.go.txt](mcp_bridge.go.txt), [store_probe.go.txt](store_probe.go.txt).
Copy them to temporary `.go` files beneath this repository before rebuilding;
their imports intentionally use the production internal packages. The service
binary runs the builder's `TestFrictionServiceProcess` *only as service bootstrap*
(register production handlers, listen on temporary socket, close on stdin EOF).
All behavioral assertions above are independently authored in
[probe.py](probe.py), not inherited from the builder's acceptance test.

## Limits

No installed binary, live daemon/store, production lifecycle, MCP host settings,
hook, skill, remote service, deployment, push, or instruction file was modified.
The MCP check uses the production server with a direct socket adapter; actual host
registration and default CLI daemon auto-spawn are not rollout-tested. Full
daemon constructor registration/shutdown wiring was inspected, while the dynamic
service test deliberately runs only the friction handlers. Process-kill recovery
is demonstrated; power-loss, filesystem corruption, and broad performance testing
are not. No Go race-detector run was attempted because this task requires no CGO.

The parent separately reports full `CGO_ENABLED=0 go test ./...` and `go vet ./...`
passing after the correction. This verdict relies on the independent evidence
above in addition to those parent checks. The next three real-task outcomes,
approved instruction corrections, and any production rollout remain future work;
the frozen memory admission backlog remains frozen.
