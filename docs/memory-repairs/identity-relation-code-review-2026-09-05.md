# Independent bounded regrade: PASS

Exact candidate: `393eeec79f80d3b4becff276c4fcffd71fa68ac5`, including the whole-relation identity guard from `8c2a05d`. Reviewed 2026-09-05 in a new git archive. This is a code and house-rule disproof gate for the relation guard plus fallback evidence preservation. It is **not** full-goal certification, live-store cleanup approval, deployment evidence, or a second consecutive full grading round.

The concrete loss reported in `/tmp/scry-identity-independent-sep05.vUECnl/REVIEW.md` is fixed. Independent probes found no remaining violation within this scope.

## Original failure: differential and restored snapshot

The identical two-episode fixture was independently rerun across all three relevant states:

| Code | Incoming routing evidence |
| --- | --- |
| Parent `5d4ff50` | Stored separately as unsafe `same_as`; text/raw preserved |
| Rejected `8c2a05d` | Merged into unrelated existing `related_to`; incoming text/raw lost; test FAIL |
| Corrected `393eeec` | Stored separately as `related_to` at the supplied timestamp; text/raw preserved; test PASS |

The source Badger backup was independently restored again into fresh review-owned stores. Source path: `/tmp/scry-childscribe-live-sep05.XlzT6P/source.badger`. SHA-256 before/after: `b8fda9a1c446d9f03dd8bc3116d1e49fd6020f7cd0e7d728553087b6ae445197`.

In `replica-4082920376`, the actual current `scry related_to hermes-ops` measurement fact remained unchanged, including its original sentence, raw `measured` relation, time, confidence, and both original episode IDs. The same replica-only routing probe that disappeared under `8c2a05d` now became a separate `related_to` fact with:

- Raw relation `aliases_index_to`.
- Exact supplied sentence `INDEPENDENT REPLICA-ONLY CASE: the audit's reviewed alias index routes to the distinct Hermes ops project.`
- Valid-from `2026-09-06T00:00:00Z`, confidence 0.93, and exactly the incoming episode ID.
- `FactsAdded=1`, `FactsMerged=0`; total facts 79,692 → 79,693.

A separate fresh-endpoint Apply probe on `replica-736837697` preserved every field on the new fact and left all 79,692 original facts plus 30,231 original entities unchanged.

## Independent adversarial results

New tests live in `new/internal/memory/resolve/independent_fallback_regrade_test.go` and `new/internal/memory/queue/independent_fallback_queue_test.go`. All passed:

1. **Current/historical exact-key conflicts:** same endpoint/relation/time with different evidence returns typed `ErrFactConflict`, zero returned stats, no mutation. A timestamp expressed in another timezone but the same instant also conflicts, matching actual UnixNano key identity.
2. **Backfill target conflicts:** exact current restatements attempting to move onto either an occupied current or historical key refuse atomically. No existing fact is deleted, overwritten, or assigned an invented timestamp.
3. **Whole-episode rollback and repeated failures:** conflict episodes first attempted entity creation, existing entity/alias updates, and a successful merge that would change another fact's confidence/provenance. Comparing full facts, entities, alias claims, and episodes showed identical before/after hashes after each of two retries. No completion marker appeared.
4. **Match beyond first triple:** an exact restatement matching the second current fallback statement merged only into that statement. The unrelated earlier statement stayed byte-equivalent. A same-text, different-raw claim was stored separately.
5. **Literal values:** `46 GiB` and `46 gib` with the same sentence/raw relation were preserved separately at their supplied times, despite normalized attribute-key equivalence; literal value equality is required for coalescing.
6. **Same-episode cases:** exact duplicate claims coalesced without duplicate episode IDs; exact duplicates with earlier supplied times retained the earlier time and maximum confidence. Different sentences or different raw relations at the same occupied key refused the entire episode. Different sentences with different supplied times both survived.
7. **Supersedes:** an exact raw reference invalidated only its matching fallback statement, preserving the other statement and all other original fields. Multiple matching raw statements returned `ErrFactConflict` with full rollback. An unknown raw reference was a no-op. Canonical `related_to` references against empty stored RawRelation and status-to-established-identity fallback references both worked correctly.
8. **Queue processing:** a real resolver conflict flowed through `Worker.process` using a deterministic local fake extractor. The original durable pending payload, source metadata, CWD, hints, Force flag, and timestamps survived; only failure bookkeeping changed. It parked immediately with the typed conflict recorded, no episode marker, and unchanged old facts. Manually retrying the unresolved fixture input parked it again with incremented attempts and preserved text.
9. **Idempotent success retry:** repeating an already completed episode returned zero stats and identical fact/entity/alias/episode state.
10. **Nonfallback control:** ordinary deployed-on restatement behavior was unchanged. Broader existing regression tests also passed.

The queue test's first draft compared Go's in-memory monotonic clock component against serialized durable time and failed on that test-only mismatch. The final test compares durable input against durable output. No product change was requested or made for that mismatch.

## Relation boundary and existing-data diagnostic

The original independent 2,154-case corpus still passes, covering all 31 effective explicit identity synonyms, existing and nested tense/modality prefixes, negation, recursive passive/preposition attacks, and nonidentity operation compounds. The canonical vocabulary remains exactly 39, identical to the parent.

Fresh diagnostic replica `replica-2106925565` contains 79,692 facts (71,968 current, 7,724 invalidated), 30,231 entities, and 8,946 distinct effective raw relations. The old→new pure mapping delta is unchanged: 19 raw strings, 25 current facts plus 2 invalidated facts, all `same_as,false` → `related_to,false`. The emitted `live-delta.json` is byte-identical to the first independent report. All original changed sentences were inspected; none proved a true identity regression under the documented whole-synonym boundary.

This remained a read-only diagnostic over existing facts. No historical relation rewrite, repair, entity fold, or alias inference was performed.

## Code review and practical limits

`matchingFactForMerge` compares endpoint/literal value, sentence, and raw relation for fallback coalescence. The original generic triple merge remains for other canonical relations. `requireVacantFallbackKey` examines both current and invalidated facts using the store key's UnixNano identity before inserts and earlier-time moves. Both checks run inside Apply's existing atomic transaction. Phase B repeats the exact-evidence lookup for same-episode duplicates. Supersedes fallback matching requires an unambiguous effective raw relation. Queue classification preserves deterministic conflicts for review.

The implementation introduces no store-wide owner inference, bulk migration, historical rewrite, timestamp nudge, new schema, telemetry, model dependency, transcript retention, or alias-transfer path. The normal identity guard remains pure and fail-closed to fallback. This addresses the reported evidence-loss case without changing the 39-relation vocabulary.

The deliberate operational tradeoff is that distinct fallback claims can now produce multiple current facts per canonical triple; claims competing for the exact same time/key park the entire queued episode for review. This review verifies preservation and refusal behavior. It does not measure the live parked backlog, real-sweep throughput, remember p95, recall scores, final live alias ownership, or the two-sweep goal conditions. Existing unsafe `same_as` facts still require separately reviewed repairs.

## Commands and artifacts

Review root: `/tmp/scry-fallback-independent-sep05.C4aYAp`.

- `go test ./...` on the candidate archive plus the original independent probes: PASS, exit 0 (`full-suite-original-probes.log`).
- `go test ./internal/memory/resolve ./internal/memory/queue -run '^TestIndependent' -count=1 -v`: PASS, exit 0 (`adversarial-tests-final.log`). This includes the additional independent probes above and fresh restores.
- `go vet ./...`: PASS, exit 0 (`vet.log`).
- `go test -race ./internal/memory/resolve ./internal/memory/queue -run '^TestIndependent(Regrade|Actual)' -count=1`: PASS, exit 0 (`race.log`).
- Parent differential: `differential-parent.log`, PASS. Rejected-commit differential: `differential-8c2a05d.log`, expected FAIL. Corrected-commit case is in the final adversarial log.
- `adversarial-tests-final.log` SHA-256: `3d77c15284cf8ebb69eea24a7a2d7d7e1f52128d878f6ea685380d52278a4148`.
- `live-delta.json` SHA-256: `b952f539089814c5564d0206eadbd4a9f7185ace827f31f3570118570053903f`.

All modifications and writes were confined to review-owned temporary archives, fixtures, reports, and independent restores. No models, remember calls, deployments, live-store writes, shared repository edits, or other reviewers' replica writes occurred.
