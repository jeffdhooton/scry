# Independent exact historical-address disproof

Verdict: **scoped source-safety PASS; no additional assertion/provenance loss or unintended operational regression was proved.** This applies only to the exact-address branch at the two new check points. It does not certify the complete memory-quality goal, all historical mutations, integration, migration, recovery, deployment, live compatibility or recall. The inherited same-episode supersession counterexample below remains outstanding.

The untouched full suite passed independently with `CGO_ENABLED=0 go test ./... -count=1`. All independent fixtures passed their stated preservation/refusal checks. The separately labeled boundary fixture passes by reproducing existing undesirable behavior on both candidate and baseline; that is not approval of that behavior.

## Scope and isolation

Read the active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, the complete private proposal, and `docs/memory-repairs/restatement-bridge-independent-rejection-2026-09-06.md`. Ran the required `scry memory orient --cwd .` first. This is the goal's fresh-context independent disproof task. No sub-agent was spawned.

Candidate: `/tmp/scry-historical-address-sep06.Mbi8Rn`, exported from ac2e166. Review copy: `/tmp/scry-historical-disproof.XiyzHG`. Separately exported baseline: `/tmp/scry-historical-baseline.DEkRP8`.

All build/test commands ran in those private directories. New tests and this report were written using apply_patch; gofmt formatted the tests. No candidate implementation file or existing test was edited. No shared source, live graph, daemon, external provider, real queue, credentials, configuration, backup, recovery or deployment was accessed or changed. Existing queue tests use their own temporary fixture stores/fake extractors. Synthetic raw tests open only test-created Badger directories after closing their Store handle. No real source content appeared in the test data or report.

The shared checkout's HEAD changed from a6a5d612abb53dbc4780c321d3d6d57adfeaaad7 to 8cd78bcdfc81b5004bfed0760d8cee2fa10b7c4e during the parent's separate work. The existing untracked `docs/MEMORY_WORKFLOW_ASSESSMENT_2026-09-04.md` remained present. The reviewer made no shared-checkout changes; the candidate stayed pinned to its private export.

## Source pins and delta

SHA-256 values matched the task's pins before and after testing:

| Candidate relative path | SHA-256 |
| --- | --- |
| internal/memory/store/historical.go | 896ba4d7739eded14bf30df9d5f5afa23c6e2a85f86917f9e495af4dc14c7de6 |
| internal/memory/store/historical_test.go | 04773bd70e86428d94419d28bbda6505a4ac47c0126508da2d2752dfdc285742 |
| internal/memory/resolve/historical.go | 9749aef36ee99197b2b57464cb9b12c7ebd2752d1f8037b7fdade591a4bafb4f |
| internal/memory/resolve/historical_test.go | 94079d9d41d83c5f5d000b01879a57d3de6b1e6b9dcea22ba17b957c1ccd3015 |
| internal/memory/resolve/resolve.go | f8f50d572f3d719fa2f8aa9131e4f34ce2c80fd249e03c9266296e5492e1c064 |

`diff -rq` against a fresh `git archive ac2e166` found exactly the proposal, the four added historical files, and the modified resolve.go. The implementation diff in resolve.go contains only the two historical-check blocks, before Phase A current matching and near the start of Phase B for unmerged inputs. No existing test expectation changed.

Independent fixture hashes:

- `internal/memory/resolve/independent_historical_test.go`: `9e7c5f37059c6066a657a91d9dc8bf9e7676a490ae9c95f0e6619a5395fbc9dd`
- `internal/memory/resolve/independent_raw_test.go`: `ed1fd5fd4fee0d4736c807eca2f6bd77db404b976b177acbde4b68c09a7da9be`

## Results

1. **History/current coexistence:** independently seeded canonical `uses` and fallback `related_to` histories alongside a current recurrence of the same sentence and triple. Exact historical restatement unions its episode into history while preserving its nanosecond start, InvalidAt and higher original confidence. The current row, including its distinct provenance and confidence, remains exactly unchanged. The supplied fixtures separately cover confidence increase, duplicate evidence in one episode, absent date at the exact occurrence, date-only parsing and an equivalent timezone.

2. **Transaction-visible staged history:** a future exclusive target causes an earlier input to be inserted already invalidated in Phase B. A second identical input at the same address sees that staged historical record and merges with it, preserving the future target and the earlier record's end while raising confidence. This directly exercises the active-transaction read, not only preexisting disk history.

3. **Supersession still executes:** a historical restatement carrying a hint for a different current triple merges its own evidence and invalidates the hinted target. Both records retain their other content and provenance; the historical record's original InvalidAt remains unchanged.

4. **No broad out-of-order admission change:** identical current assertions arriving one hour earlier without a date still merge/backdate under the existing canonical and fallback behavior. `not-a-date`, `2026-02-30`, and a single space also retain their existing fallback behavior outside an occupied historical address. Those same fixtures pass on the independent ac2e166 baseline. The candidate's supplied occupied-history malformed-date cases refuse atomically. This does not endorse permissive parsing generally.

5. **Later undated recurrence:** with historical Caldera and current Emberbank under `replaced_by`, a later undated Caldera input adds a new assertion and invalidates Emberbank, preserving the old Caldera history exactly. It does not attach the episode to the old historical interval. The same expected behavior passes on ac2e166. This avoids the earlier rejected experiment's sentence-only inference.

6. **Occupied history cannot be bypassed through a current match:** changing only the sentence, or using inverse `used_by` so normalized endpoints/relation match while RawRelation differs, refuses before coalescing onto an existing canonical current row. Facts, stats and events remain unchanged. This is an intentional exact-assertion refusal, not an unrelated-ingestion regression.

7. **Unsupported raw payload refusal:** unknown evidence fields, duplicate episodes fields, leading whitespace, reordered JSON fields and a payload/address start mismatch all refuse through Apply. Each refusal returns ErrFactConflict with a safe key digest, zero stats and zero observer events. Complete raw key/value maps before/after match, including fact rows, indexes, entities, episode markers and schema metadata. A separate unrelated dependency input succeeds beside the same unsupported historical payload and leaves its exact raw bytes unchanged. Refusal is local to the touched address, not a store-wide canonicalization requirement.

8. **Late-conflict rollback:** one episode first raises historical confidence and unions provenance in Phase A, then stages a new entity and alias, supersedes an unrelated current assertion, adds another fact, and finally collides on distinct same-address fallback sentences. The entire raw key/value map is byte-identical after refusal; returned stats and emitted observer events are both zero. This includes the absent episode marker and restores original provenance/confidence/invalidation bytes.

The unchanged full no-CGO suite also passes the existing queue outage/recovery drain test, resolver canonical coalescing/backdating expectations, fallback refusal tests and store/merge tests. This independently distinguishes the new candidate from the prior broad experiment whose unchanged suite was red.

## Inherited same-episode supersession boundary

`TestIndependentSameEpisodeSelfSupersedesBoundary` reproduces this deterministic case on both ac2e166 and the candidate:

- Empty store; episode occurs at t2.
- First input A has explicit t1 < t2 and confidence .95.
- Second input has the identical A address/sentence, confidence .2 and `Supersedes: A`.
- Phase A finds nothing. Phase B adds first A as current. Second A's historical check sees it still current and returns no match. Its own supersession then invalidates A, after that check. The existing canonical insertion path calls PutFact on the same address, reopening the row at .2.

Both versions report two additions and one invalidation, with one final current record at .2. This proves a remaining loss/invalidation weakness, **not a regression introduced by this candidate**. The candidate must not be described as protecting every historical record formed anywhere in Phase B, or as solving general direct-PutFact metadata replacement. Its exact pre-check placement has a real temporal boundary.

No implementation correction is required for the narrow no-new-regression claim proven here. If the next scope is broadened to protect history created by the same input's supersession, the smallest apparent direction is another exact-address preservation check after `applySupersedes` and before the remaining merge/add path. That is a proposed follow-up, not a tested patch; it needs its own explicit behavior contract and fresh review. Do not silently count the present check as covering it.

## Commands and observed exits

Run from the shared checkout only to orient/read/export:

```sh
scry memory orient --cwd .
git show ac2e166:internal/memory/resolve/resolve.go | diff -u - /tmp/scry-historical-address-sep06.Mbi8Rn/internal/memory/resolve/resolve.go
git archive ac2e166 | tar -x -C /tmp/scry-historical-baseline.DEkRP8
diff -rq /tmp/scry-historical-baseline.DEkRP8 /tmp/scry-historical-address-sep06.Mbi8Rn
```

The diff commands exit 1 because the expected candidate changes exist. Initial/final `shasum -a 256` commands verified all five exact candidate paths above. `cp -R /tmp/scry-historical-address-sep06.Mbi8Rn/. /tmp/scry-historical-disproof.XiyzHG` populated the separate review copy.

Run from the review copy before adding independent tests:

```sh
CGO_ENABLED=0 go test ./... -count=1
```

Exit 0. Every package completed successfully or reported no tests; memory/queue 11.504s, memory/resolve 17.159s, memory/store 15.832s and daemon 29.535s in that run. This was the full unchanged suite, including the candidate's supplied new tests.

Run from the review copy after adding/refining independent tests:

```sh
gofmt -w internal/memory/resolve/independent_historical_test.go internal/memory/resolve/independent_raw_test.go
CGO_ENABLED=0 go test ./internal/memory/resolve -run TestIndependent -count=1 -v
```

Final exit 0, package 1.517s. Earlier independent batches were also green; the raw-relation case was refined to specifically test inverse `used_by`, and the baseline boundary fixture was then added. No implementation changes were made to obtain a pass.

Copied the independent_historical_test.go fixture file to the private baseline and ran there:

```sh
CGO_ENABLED=0 go test ./internal/memory/resolve -run 'TestIndependent(Undated|SameEpisodeSelfSupersedes)' -count=1 -v
```

Exit 0, package 1.002s. This compares the benign earlier-ingestion, recurrence and inherited supersession-boundary cases directly. The complete baseline repository suite was not rerun; no claim relies on such a run.

## Limits and handoff

This review found no counterexample to the specified narrow preservation improvement. It is one independent source/fixture round, with no real backup or live replica. It grants no deployment or whole-goal PASS. Conservative refusal can intentionally park episodes that touch noncanonical historical bytes or distinct occupied assertions; the queue retains them for review.

Current-triple sentence loss/backdating, direct PutFact metadata replacement, general unsupported raw current-row rewriting, malformed dates outside history, exact supersession identity/temporal redesign, normalized target identity and legacy UnixNano/address collisions remain outstanding. No old evidence was recovered and no transcript retention was added. Keep this experiment separate from the already-reviewed schema-refusal artifact and its deployment work.
