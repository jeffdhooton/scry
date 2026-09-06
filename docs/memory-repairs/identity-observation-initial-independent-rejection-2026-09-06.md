# Immutable observation primitive: independent database/serialization disproof

2026-09-06. Verdict: **NO-GO for the exact private candidate reviewed.** A concrete provenance failure is reproduced in two independent regressions. The writer commits, and the reader returns, observations whose episode contains conflicting duplicate identity or timestamp members. This is a bounded private-unit rejection, not a claim of observed live corruption or failure of an implemented controller.

## Scope and preserved inputs

I ran `scry memory orient --cwd .` first and read its output. I read the complete active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, complete `OBSERVATION_CONTRACT.md`, and complete root review `docs/memory-repairs/admission-controller-v2-independent-review-2026-09-06.md` (67218c1c...). That review explicitly selects this uncalled primitive and excludes controller integration/adoption.

I created a new private export of baseline `87a6d1a262d5d7c62037d66335b9a020f89fd4a6` at `/tmp/scry-observation-independent-sep06.Bcn39q`. I copied only the exact candidate observation source, its supplied tests, and the contract from `/tmp/scry-observation-evidence-sep06.97oku1`, using `apply_patch`, then confirmed their hashes. The baseline already contains the separately reviewed private generation primitive and its test snapshot helpers. I authored one independent test file and this report/output evidence. Candidate production source and supplied tests were never fixed or modified. All authored/copied text files used `apply_patch`; the new test was formatted with `gofmt`.

No shared checkout edits, live graph reads/writes, provider calls, deployments, sweeps/retries, configuration changes, or external messages occurred. All database fixtures are synthetic temporary stores. Initial required orientation was the sole memory interaction. Root's later proposed correction is not present here and has not been graded by this report.

## Blocking finding: ambiguous raw episode accepted as provenance

`identity_observation.go:114` calls `GetEpisode` and compares the resulting ID and instant. Baseline `store.go:328` loads the `ep:` key and uses ordinary `json.Unmarshal`. Duplicate JSON members therefore select a later value. The new primitive relies on that lossy projection to establish provenance even though the retained episode bytes contain conflicting claims.

`TestIndependentObservationRejectsAmbiguousEpisodeProvenance`, at independent test line 501, starts from a normal synthetic episode and prefixes either an earlier conflicting `id` member or an earlier conflicting `occurred_at` member to the complete existing object. The existing expected member remains later in the object. The exact episode key still names the selected episode. No hash collision, deletion, malformed observation, or caller-discarded error is involved.

The ID fixture has the effective form:

```json
{"id":"foreign-episode","id":"episode:独立/🧭", "other original episode fields":"retained"}
```

The time fixture analogously contains an earlier timestamp one hour after the original, followed by the original exact UTC nanosecond timestamp. These snippets explain the structure; executable tests manufacture the complete actual records.

For both fixtures, the writer returns nil and commits a new `io:` row. Each regression also seeds a canonical observation independently of writer behavior, then invokes the reader; it returns one record and nil error. Thus merely fixing the write path would leave the read regression applicable. All raw episode bytes remain unchanged, so their ambiguity is not resolved by the successful operation.

Captured output in `PROVENANCE_TEST_OUTPUT.txt`:

```text
ambiguous episode id accepted as authoritative provenance: <nil>
ambiguous episode allowed committed observation
ambiguous episode id accepted on read: records=1 err=<nil>
ambiguous episode occurred_at accepted as authoritative provenance: <nil>
ambiguous episode allowed committed observation
ambiguous episode occurred_at accepted on read: records=1 err=<nil>
```

The baseline behavior is `GetEpisode`'s decoded projection; this review does not claim that existing API newly regressed. The observation primitive's new provenance guarantee is the failing boundary. A retained episode with two conflicting IDs or instants cannot establish unambiguous key/body agreement merely because the expected value was last. This violates the contract's required provenance matching/refusal and the active goal's evidence-preservation discipline.

Correction direction: inspect the raw top-level episode object within the existing transaction; require unambiguous explicit ID/time scalars and compare them to the selected key and exact observation instant. Cover duplicate and case-folded member spellings, invalid text, missing/null/wrong-typed members, and incomplete/trailing JSON. Preserve unrelated episode extensions and all original bytes; do not repair this by rewriting historical episodes or globally requiring their full JSON to match a new struct serialization. The root has proposed a private validator along these lines, but implementation and verification remain outside this frozen rejection.

## Independent attempts that passed

Seven independent top-level tests passed; the eighth contains the two failures above. All four supplied observation tests also passed. Passing results are limited to the current parsed-input contract and the synthetic cases exercised.

| Boundary | Concrete coverage and result |
| --- | --- |
| Complete parsed objects and independent occurrence identity | 145 distinct records: two episode IDs; ordinals 0, 1, 2, 9, 10, 100; same declaration name/type; true/false `TypeFallback`; nil/empty/ordered duplicate aliases; both endpoint sides; nil/empty/populated `Supersedes`; an additional conflicting description at the identical occurrence. Every returned object deep-equals its original input. No collapse. |
| Exact original fact semantics | Original value-like/inverse relation endpoints, full fact text, unparsed `ValidFrom`, next-representable confidence value, and all three supersession fields survive. Static comparison with current `extract.Ent`, `extract.Fct`, and `SupRef` found no omitted parsed fields. No claim of recovering pre-parser invented types or original transcripts. |
| Encoding refusal and UTC | 27 invalid cases include each text field and nested supersession field with invalid UTF-8, invalid kinds/side/version/ordinal, missing payloads, zero and out-of-range time, NaN and infinity. Accepted Unicode includes composed/decomposed strings, emoji, control characters, quotes, HTML-sensitive characters, and Unicode line separators. Offsets -3599, -1, 0, 1, 86399 seconds produce identical canonical bytes/keys for the same instant. Monotonic time normalizes successfully. Supplied test separately distinguishes adjacent nanoseconds. |
| Occupied raw collision and address validation | Unknown extension, duplicate member, whitespace/noncanonical JSON, malformed bytes, different canonical revision deliberately installed at the candidate address, wrong address, wrong episode body ID, wrong instant, and missing episode all refuse reads without changing bytes. Applicable occupied writes refuse without changing bytes. Canonical candidate bytes are compared with occupied bytes, not accepted solely from matching digests. |
| Caller ownership | Supersession pointer and fact fields changed after staging do not change committed bytes; mutations of returned supersession pointers do not affect durable data or later reads. Supplied test separately changes the input alias slice after staging and proves owned bytes. |
| Transactions and staging | Returned callback error, panic, and same-transaction episode rollback preserve the complete prior map and emit zero events. Actual `badger.ErrTxnTooBig` arises from the primitive's `Set` after a 190000-byte filler and a 190000-character observation, using a 2 MiB memtable and 200000-byte value threshold; no mock error injection. All staged bytes roll back. A same-transaction episode plus observation commits successfully, with exactly the episode's single event. Nil/nontransactional/escaped write facades refuse. |
| Pagination and complete encoded budget | All 145 inputs are traversed without loss, duplicates, or wrong-episode results using count limits 1/3/100 and byte budgets 1100/1600/10000. Actual `json.Marshal(observationPage)` bytes stay within each bound, including the full continuation key and envelope. Keys advance in full-byte lexical order across non-numerical ordinal ordering. Exact cursor-inclusive boundary succeeds; one byte less refuses visibly. Missing, foreign, fabricated prefix-only cursors and invalid count/byte bounds refuse. A small row before a large row is returned; repeated continuation at the large row returns the specific error and exact key digest, with no advancing cursor. |
| Raw backup/restore/replay | A full raw map containing observations and deliberately opaque bytes in en:/al:/att:/ig:/iga:/fa:/adj:/future-family: survives Backup, direct Badger Load, direct loaded-map inspection, close/Open, and identical observation replay exactly. The restored structured endpoint observation deep-equals the original. Unknown families are not rewritten or deleted. |
| Other-family/routing/event isolation | The 145-record fixture changes only new io: keys and preserves every preexisting byte. Identical replay is an exact raw-map no-op. Zero observation events, zero entities, zero facts, and no alias lookup claim are observed. Static reference search confirms only the private primitive and tests refer to its writer/reader; no production resolver, recall, admission, or generation selector calls it. |

The reader conservatively reserves a continuation key before finding that a page is terminal; the eventual terminal response can be smaller than the required trial budget. The tests measure the explicit full cursor-inclusive budget and do not claim optimal packing. This review does not elevate that conservatism to a correctness failure. Records larger than the maximum page remain visibly refused; no useful production detail/chunk interface exists here.

## Executed verification

From the independent directory:

```sh
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentObservation|TestObservation' -count=1 -v
CGO_ENABLED=0 go test ./internal/memory/store -run TestIndependentObservationRejectsAmbiguousEpisodeProvenance -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

The first focused execution initially needed a grader-only compile correction: an invented `EntityNames` test call was removed; no candidate source changed. The focused tests then passed everywhere except the two provenance subcases. I extended those same failing cases to independently assert read refusal and froze the regression file. A first full suite with write assertions had the same two failing subcases; the final full suite ran the exact frozen write-and-read test file listed below. `FULL_SUITE_OUTPUT.txt` preserves the complete final stdout/stderr. Exit status is 1, with only `TestIndependentObservationRejectsAmbiguousEpisodeProvenance/{id,occurred_at}` failing. All other packages passed or reported no test files; store took 18.292 seconds and daemon 27.901 seconds. No test result used a cache.

## Exact pins

| Artifact | SHA-256 |
| --- | --- |
| Rejected `internal/memory/store/identity_observation.go` | `56f7ce81f3a7542084e296e1356fc9525fe0d8682faf1959b454fb45f56b6b55` |
| Supplied `internal/memory/store/identity_observation_test.go` | `e52ecb3025494ea3f334d62fbd887e23b435e6c442141d2af451c85b9d724712` |
| Frozen `internal/memory/store/observation_independent_disproof_test.go` | `8cf067486ffc704fbaba9eacdceee05b4f3fda075bd48d2acf7d719f79c8e113` |
| `OBSERVATION_CONTRACT.md` | `3f16ec0d09d953b6735aa60e4345c9812f3f84fcb68707a162d206e4383a77ed` |
| Root controller V2 independent review read | `67218c1c362ab20f8a0d328f3a1895935b132c76c33c1e31c5a8f955a3fde49b` |
| Baseline `internal/memory/store/store.go` | `4b18a0037534aa0111b3d8fede8883dbf4bfa3ffe955b3ff15ebd0d07fa38f88` |
| Baseline private `internal/memory/store/identity_generation.go` | `e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592` |
| Baseline `internal/memory/extract/extract.go` | `521904fd2881e446b7d175f7f0a0869a1ec352e41970c7b4a31fe6dafa1b9067` |
| `PROVENANCE_TEST_OUTPUT.txt` | `3ef9737e126b3997fe9b359226bc9375dcd1c6a7c47f85342ef9dec17a18a56d` |
| `FULL_SUITE_OUTPUT.txt` | `b32bfe7d8ae19f03dd1e635a05221562480bf74c3bea049644b6f577da32840c` |

The report's own hash is delivered separately after writing. The rejected source and both failing regressions remain preserved. A later correction requires its own exact-source review. Even a later bounded PASS cannot certify resolver occurrence capture, materialized-state/disposition links, useful user-visible inspection, outermost finalization, admission/adoption, all-writer coverage, live graph cleanup, or any whole-goal completion clause.
