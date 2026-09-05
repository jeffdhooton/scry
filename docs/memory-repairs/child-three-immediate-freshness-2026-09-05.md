# Independent immediate freshness gate — 22:35:50 UTC

**Bounded PASS** extending the exact three-alias gate from the complete 22:31:22 source to `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-child-immediate-223550.badger`, independently verified SHA-256 `89bfa426f1ae6955fe7cb04fe9bb0fa8472e5e7f089b8cd5e46386fa524eca67`.

I directly loaded that complete backup into a new private Badger replica and compared every raw key/value byte against the independent original 22:31 restore. Both have 242,732 keys. There are **exactly two modified keys, zero additions, and zero removals**:

| Key | 22:31 source value | 22:35:50 source value |
|---|---|---|
| `meta:last_sweep_at` | `2026-09-05T22:24:11.984441Z` | `2026-09-05T22:34:14.354989Z` |
| `meta:last_sweep_report` | `Jeffs-MacBook-Pro.local`; 2,352 files scanned, 8 ingested, 11 episodes, 0 errors; finished 18:24:11.984441−04:00; Claude 4 / Codex 7 episodes, Claude 1 / Codex 7 files | `Mac.attlocal.net`; 94 files scanned, 0 ingested, 0 episodes, 0 errors; finished 18:34:14.354989−04:00 |

The newer report omits per-source maps because the recorded sweep ingested nothing. The exact original/new bytes and individual SHA-256 values are retained in `evidence-223550/freshness.json`; this report does not normalize away the host or omitted fields.

Every other one of the 242,730 key/value pairs is byte-identical. In particular there are **no cursor changes**, no pending payload/scheduling/attempt changes, no new source episodes or facts, no alias/attestation/rejection changes, and no semantic metadata change. The complete 22:31 semantic review therefore remains applicable without assuming that equal entity/fact counts or Expected alone imply complete closure.

Candidate Store.Open preserves the direct restored raw state. The exact manifest preview is ready, touches 2,106 complete current/history Child facts, and makes zero raw writes. It retains the reviewed plan `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`, Child entity fingerprint `b7fa1df19b48d1713368d546bda53d32ea6e748292092936fc10b55f26026c59`, and touching-fact fingerprint `5d879da1900f8e146b5e98858efa44e77445c0ad21b5b21b20b130badb1913fa`. Listings, claims, rejection closure and exact manifest remain valid.

Reproducer: `/tmp/scry-child-three-fresh-independent.G20a7Q/internal/memory/store/immediate_223550_test.go`; command `go test ./internal/memory/store -run '^TestImmediate223550Independent$' -v -count=1`, PASS, 1.720 seconds including package result. Private immediate replica: `evidence-223550/direct/`. Comparison source: `evidence-223122/direct-restore/`. Prior full report `REVIEW-223122.md` is unchanged, SHA-256 `f6c4739f9e80bbea98d52b8cbe247ef7c0cf7408464858e977232b7ef1631494`.

The actual live post-apply prediction must start from this immediate state, retaining these two newer sweep metadata values, then make only the seven reviewed repair-key changes. All immediate backup/locked-input/deployed-writer checks and independent actual post-apply raw/lookup/second-no-write/five-suite grading remain required. The first three live markers still prohibit an old-binary-only downgrade. This is no approval for other alias repairs, fact edits, rehomes, retries, or whole-goal completion.

No live writes, deployment, provider call, queue action, or shared source edit was made. This task performed a fresh full raw comparison and preview only; it did not apply the repair to this immediate replica or repeat the already passed 22:31 complete apply experiment.
