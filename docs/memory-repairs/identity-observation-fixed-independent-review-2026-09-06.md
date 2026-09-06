# Corrected immutable observation primitive: independent review

2026-09-06. Verdict: **bounded PASS for exact private, uncalled source 83a39c03... under the updated explicit provenance contract.** The two preserved independent rejection regressions now pass unchanged, including separate writer and reader assertions. All new parser probes, all previous preservation/transaction/paging/restore tests, and the full noncached no-CGO suite pass. This does not certify a controller, resolver integration, adoption, useful user-visible inspection, or the whole memory-quality goal.

## Scope and evidence preservation

This is the independent rejecting reviewer's follow-up on a correction built by the root. I had already run and read the required initial `scry memory orient --cwd .`, complete active goal, and complete controller V2 independent review. For this correction I read the complete updated `OBSERVATION_CONTRACT.md`, complete exact corrected source, and complete added builder provenance tests before testing.

I created a NEW private export of baseline `87a6d1a262d5d7c62037d66335b9a020f89fd4a6` at `/tmp/scry-observation-corrected-independent-sep06.3rRHua`. I copied the exact corrected source, updated contract, unchanged supplied observation tests, builder's added provenance tests, and my entire frozen prior regression file using `apply_patch`. I authored only one additional independent test file plus report/output artifacts; `gofmt` formatted the new independent file. No implementation changes were made by this grader.

The rejected export at `/tmp/scry-observation-independent-sep06.Bcn39q` remains untouched. Its rejected source hash is still `56f7ce81f3a7542084e296e1356fc9525fe0d8682faf1959b454fb45f56b6b55`, its frozen regression hash is still `8cf067486ffc704fbaba9eacdceee05b4f3fda075bd48d2acf7d719f79c8e113`, and its report hash is still `4ad7054893a06cc595a08abda584bb86c3c1fe49b67cd8103dfa6d9331d52fad`. Those original failure results are not erased or relabeled.

No shared checkout edits, live graph access/writes, provider calls, deployment, sweeps, extraction retries, configuration changes, or admission integration occurred. All fixtures are synthetic temporary Badger stores. No new memory interaction occurred during the correction review.

## What changed and why the original disproof no longer holds

The source diff against the preserved rejected candidate is confined to adding the `io` import and replacing `observationProvenance`. The original codec, immutable writer, cursor handling, and page-budget code are unchanged.

The rejected function relied on baseline `GetEpisode`'s ordinary `json.Unmarshal` projection, which selected the last duplicate member. The correction reads the actual `ep:` value in the current transaction and tokenizes its top-level object. It requires exactly one case-insensitively recognized `id` and `occurred_at`, checks their scalar JSON encoding and exact identity/instant, requires complete valid UTF-8 JSON with no trailing value, and rejects ambiguous known fields. It does not reserialize or write episode bytes. Unrelated values are consumed as raw JSON; they are not decoded into routing or identity evidence.

`TestIndependentObservationRejectsAmbiguousEpisodeProvenance/{id,occurred_at}` is byte-for-byte unchanged from the rejected review. Each previously committed an observation against a conflicting duplicate episode member, then separately seeded canonical observation bytes and proved that the reader also accepted them. Both now refuse with `errIdentityObservation`, preserve all prior raw bytes, and pass. No assertion was weakened and no failing test was removed.

The updated contract explicitly limits accepted known scalar representations. Equivalent escaped ID strings, redundant timestamp precision, and other noncanonical scalar encodings may be refused. This intentional fail-closed behavior is compatible with this private unit; it is not approval of a broad old-episode migration or proof that all historical episode representations will be accepted. Whole-episode whitespace, field order, unrelated extensions, and an exact instant expressed in a canonical nonzero minute offset remain valid controls.

## New independent parser and preservation probes

`observation_provenance_independent_correction_test.go` adds two tests. The parser matrix has 41 cases: 9 accepted controls and 32 refusals. Every refused fixture is tested through both the writer and an independently seeded canonical observation read. Refusals preserve the full raw map. Accepted fixtures preserve exact episode bytes, yield the complete original observation, replay without raw changes, and emit zero graph events.

| Boundary | Independent evidence |
| --- | --- |
| Duplicate matching | Repeated identical IDs/times still refuse. Escaped duplicate `id`, escaped uppercase duplicate `Occurred_At`, and a conflicting known field after the expected one refuse. Single mixed-case fields and escaped field names accept. |
| Object framing and text | Array/null/string roots, empty object, missing/wrong closure, trailing comma, trailing null/garbage/incomplete object, non-JSON trailing Unicode whitespace, BOM, invalid UTF-8 in an unrelated field name, and malformed unknown nesting refuse. Ordinary leading/trailing JSON whitespace and reordered fields accept. |
| Known scalar types and exact values | ID objects/arrays/booleans and time null/objects/arrays/booleans refuse. Invalid date and a one-nanosecond mismatch refuse. Canonical scalar policy rejects redundant time precision, escaped time suffix, comma fractional delimiter, and equivalent escaped Unicode ID. A canonical +05:45 time representation of the same instant accepts. |
| Unknown extension controls | Duplicate unknown fields, nested conflicting `id`/`ID`/`occurred_at` members, an unknown numeric token `1e99999`, and unknown strings/null/arrays accept unchanged. Nested names do not become top-level provenance. Raw numeric extensions are not forced through a float64 conversion. |
| Current transaction | A raw episode with mixed-case provenance and nested opaque duplicate IDs is staged in AtomicWrite. Observation write/read see it in that transaction; returning a synthetic error rolls the entire map back. The corresponding successful transaction commits the original bytes. |
| Extension backup/replay | The accepted opaque episode plus observation survives full Backup, direct Badger Load and immediate raw-map comparison, close/Open, identical replay, and structured read with complete byte/object equality. |

All 10 cases in the builder's separate provenance test also pass. They add missing ID/time, null ID, numeric time, uppercase duplicate members, invalid UTF-8 in an unknown value, trailing object, noncanonical ID escape, and a nested-extension control. Builder tests supplement, rather than replace, the independent cases.

## Unchanged complete-unit regressions rerun

All eight prior independent top-level tests and all four original supplied observation tests pass on the correction:

- 145 distinct occurrence/revision records across two episodes, ordinals 0/1/2/9/10/100, repeated names/types, conflicting descriptions, true/false fallback flags, nil/empty/ordered duplicate aliases, both endpoint sides, and nil/empty/populated Supersedes preserve complete parsed fields without collapse.
- Invalid UTF-8 across all current parsed text fields, malformed occurrence shape/version/ordinal, zero/out-of-range time, NaN/infinity refuse. Accepted Unicode and exact UTC instants survive, including second-precision source offsets and adjacent nanoseconds. Supersession and caller alias/pointer mutation do not alter staged bytes; returned pointers do not mutate storage.
- Unknown/duplicate/noncanonical/malformed occupied observation bytes, forced raw collision, wrong key, mismatched/missing episode provenance refuse unchanged. Identical replay is a raw-map no-op; changed content at an occurrence remains a separate revision.
- Callback error, panic, expired facade, and actual primitive `badger.ErrTxnTooBig` staging failure preserve all prior bytes and emit zero observation events. Same-transaction episode success and rollback remain correct.
- Paging traverses every one of the 145 records without wrong-episode results, skips, duplicates, or stalled continuation using limits 1/3/100 and byte budgets 1100/1600/10000. Full marshaled page bytes include envelope and exact continuation key and stay bounded. Exact cursor-inclusive budget succeeds; oversized rows visibly refuse with key digest and never advance past the row.
- Complete raw Backup/directLoad/Open/replay equality includes opaque en:/al:/att:/ig:/iga:/fa:/adj:/future-family: bytes. Only io: keys are added by the observation primitive. Entity/fact counts stay zero, alias lookup gains no claim, and observation writes emit no events.

Static scope is unchanged: no production resolver, recall, generation selector, or admission path calls the private observation reader/writer. The corrected primitive still stores only parsed original input; it neither creates support nor removes graph identities. Existing baseline resolver zero-fact behavior was not changed and is included in the passing full suite.

## Executed commands and results

From this new independent export:

```sh
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependent.*Observation|TestIndependentCorrectedProvenance|TestObservation' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

Both exited 0. Focused run: store 3.049 seconds. Full run: store 23.216 seconds, resolver 18.425 seconds, daemon 30.032 seconds; every package passed or reported no test files. No cached test result was used. Complete outputs are preserved in `FOCUSED_TEST_OUTPUT.txt` and `FULL_SUITE_OUTPUT.txt`. No further source or test edits occurred after these passing checks.

## Exact hashes

| Artifact | SHA-256 |
| --- | --- |
| Corrected `internal/memory/store/identity_observation.go` | `83a39c03435185963a4a2931f94f30f03a7bd1a86c6e4c545ece95580ad9e0ab` |
| Original supplied `internal/memory/store/identity_observation_test.go` | `e52ecb3025494ea3f334d62fbd887e23b435e6c442141d2af451c85b9d724712` |
| Builder `internal/memory/store/observation_provenance_test.go` | `4249bac10573ece0476f3006ffe6fea2d8f1652b663f234f9a64a3b435f8b442` |
| Unchanged independent `internal/memory/store/observation_independent_disproof_test.go` | `8cf067486ffc704fbaba9eacdceee05b4f3fda075bd48d2acf7d719f79c8e113` |
| New independent `internal/memory/store/observation_provenance_independent_correction_test.go` | `08068b7dea4e529fe2425d8953a794e9a781e0754a2aa7fccc33fb646618d78d` |
| Updated `OBSERVATION_CONTRACT.md` | `43cd045132789fe15d440ad343dfe0de6900351ed2b6e729883be56a3d0c99c1` |
| `FOCUSED_TEST_OUTPUT.txt` | `36e25fcca5d3986178768a96d02671d17535f9a2967ef66db61d904928802cc6` |
| `FULL_SUITE_OUTPUT.txt` | `c99c885ddcba0c453a21bde1abf749ca4ff94c312e0a56cee07f6febe6524d1b` |

The report's own hash is delivered after writing. The baseline store and generation source are unchanged from export 87a6d1a and the previous review.

## Limits of this PASS

The reader still uses a conservative cursor-inclusive trial budget before discovering a terminal page; optimal packing is not claimed. Oversized records remain explicit refusals with no production detail/chunk surface. Bounds describe the encoded private page, not future RPC wrappers, parser memory, or CPU limits. Pagination tests use stable synthetic snapshots; no concurrent insert snapshot protocol is certified.

The caller must propagate returned failures through the owning transaction. This primitive does not poison a caller that deliberately ignores errors or validate later caller mutations at an outermost finalization boundary. Resolver occurrence capture, materialized-state/disposition links, support policy, generation vote buffering, compulsory finalization, useful CLI inspection, all-writer lifecycle enforcement, adoption, live replica grading, and graph cleanup remain separate unapproved work. This one corrected bounded verdict is not two whole-goal grading rounds and does not authorize deployment or declare the user's goal complete.
