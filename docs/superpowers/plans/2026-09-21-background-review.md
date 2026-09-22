# Background Review Implementation Plan

**Goal:** Install a working, bounded background review service in Scry.
**Architecture:** Separate snapshot/evidence, reviewer, and scheduler/storage modules. Local RPC/CLI/MCP adapters expose provisional snapshot-bound results.
**Tech Stack:** Go 1.26.2, existing daemon/RPC and Cobra, standard HTTP and JSON; no new runtime dependency.
**Spec:** ../specs/2026-09-21-background-review-design.md

## Global constraints

Default off; local code daemon only; one review at a time; explicit repos and provider; bounded durable daily request reservations; no edits or memory admission; evidence exclusion and freshness are enforced.

## Tasks

- [x] Snapshot and reusable evidence: `internal/review/snapshot.go`, types and tests. Prove real Git states and exclusions before implementing capture.
- [x] Reviewer: `internal/review/provider.go` and tests. Use fake HTTP servers to verify bounded, tool-free requests and strict cited output parsing before implementation.
- [x] Service: `internal/review/service.go`, `store.go`, config and tests. Test persistent reservations, debounce, duplicate suppression, stale results, restart and cancellation.
- [x] Integration: local daemon lifecycle and evidence enrichment, CLI, MCP, profile tests, README and setup routing instructions.
- [ ] Validate and install: focused tests, race tests, full tests, vet, independent review, local binary rollback/install, existing launchd restart, real CLI/MCP/provider checks and memory record of outcome.

## Interface contract

Package `review`:

```go
type FileState struct { Path string; SHA256 string; Mode string }
type Evidence struct { ID string; Kind string; Path string; Content string }
type Snapshot struct {
 ID string; Repository string; Head string; CapturedAt time.Time
 Files []FileState; ChangedFiles []string; Evidence []Evidence; Warnings []string
}
type CaptureOptions struct { MaxBytes int; Exclude []string }
func Capture(context.Context, string, CaptureOptions) (Snapshot, error)
func Fingerprint(context.Context, string) (string, error)
type Finding struct { Severity string; Title string; Detail string; EvidenceIDs []string }
type ReviewOutput struct { Summary string; Findings []Finding; TestGaps []string }
type Reviewer interface { Review(context.Context, Snapshot) (ReviewOutput, error) }
type ProviderConfig struct { Protocol, BaseURL, Model, APIKeyEnv string; MaxOutputTokens int }
func NewProvider(ProviderConfig) (Reviewer, error)
```

Snapshot capture is independently implemented with tests; provider and adapters can proceed against this agreed contract. Root owns service/store/config/daemon integration. All contributors preserve other changes.

RPC methods `review.status|preview|run|list|get`, parameters `repo` for preview/run/list, `id` for get; no model calls in status/preview/list/get. `run` returns queue state; `get` returns a record with full captured evidence and freshness. CLI adapters encode JSON; MCP forwards validated arguments to local daemon. Status reports effective config without keys, usage, and per-repo activity.
