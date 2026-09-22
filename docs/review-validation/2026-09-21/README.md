# Background review validation — 2026-09-21

## Acceptance scenario

A real temporary Git repository contains a permission check and three callers:
member, admin, emergency service. Its indexed references and current caller
source are combined with an attributed decision requiring emergency access
without member/admin credentials. The regression removes that exception.

The deterministic test verifies:

- No inference before the quiet checkpoint, including a new edit resetting it.
- All three caller references/current source and the prior decision reach the reviewer.
- Findings retain snapshot/context identity, duration and returned usage.
- A foreground-work window can overlap the review.
- Two independent RPC clients retrieve the same completed provisional record.
- A supporting memory change and a caller-source change both invalidate freshness.
- The memory service receives reads and zero writes.

This fixture responder validates plumbing, not model reasoning or production accuracy.

## Real provider attempt

[Full captured record](regression.json): `glm-5.3-flash` on the already configured
Z.ai Messages endpoint returned **HTTP 429**, after 714 ms. One request was made;
there was no automatic retry and no alternate-provider attempt. No review text
or token usage was returned, so cost and model-quality results are unknown.
The live regression test failed visibly. The clean-control model test was not
run because the provider was blocked. Model-quality acceptance remains pending
provider availability.

The installed service records that block and requires an explicit
`scry review resume` after availability is fixed. The original failed request
and its reservation are preserved; restarting cannot clear the block or usage.

## Repeatable validation

```sh
CGO_ENABLED=0 go test ./...
go test -race ./internal/review ./internal/daemon ./internal/mcp ./cmd/scry -run 'Test(Service|Snapshot|Review)' -count=1
CGO_ENABLED=0 go vet ./...
```

The live test and control commands are documented in
[BACKGROUND_REVIEW.md](../../BACKGROUND_REVIEW.md). Ordinary tests do not make
paid provider calls.

## Real-work measurements to collect

For each actual foreground task, retain review ID, foreground start/finish,
review completion, whether useful findings arrived before completion, human
verdict (useful/false positive/inconclusive), time spent reading/interrupted,
provider token usage, and billed or rate-estimated cost. A fixture result is
not a production recall, precision, latency, or interruption-cost benchmark.
