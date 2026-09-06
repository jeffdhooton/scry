# Immutable parsed input revision — bounded independent disproof

2026-09-06. PASS for the frozen private input-revision contract. No contract violation was found in source review or the independent synthetic tests. This verdict certifies no production routing, complete current result, support classification, lifecycle protection, live-store change, recall performance, or whole-goal clause.

The independent export is `/tmp/scry-input-revision-disproof.s8T9zi`, made with `git archive` from baseline `8fda77e3fd4d4f001ae9038994ab6e12d8bab480`. I copied exactly the five pinned candidate files from `/tmp/scry-input-revision-sep06.UfziK3` using apply_patch and verified every hash. Candidate source and supplied tests remained unchanged. My only authored additions are the independent synthetic test, logs, this report, and its hash manifest. No candidate/shared source, real graph data, service, provider, config, room, persistent memory, sweep, deployment, or live store was changed.

I ran and read `scry memory orient --cwd .` first, then read the complete active goal objective, complete INPUT_REVISION_CONTRACT.md, complete current episode design review, observation/outcome/adoption/owner contracts and their baseline implementations, the generation transaction/read guards, AtomicWrite/view implementation, and all supplied input-revision tests. The adopter diff against baseline is exactly one additional reserved family, `io-input:`.

## Executed evidence

`CGO_ENABLED=0 go test ./internal/memory/store -run 'Test(IndependentInput|InputRevision)' -count=1 -v`

PASS, store 2.411s: six independently authored top-level tests plus all nine unchanged supplied top-level tests. The complete final output is `TARGETED.log`. An earlier successful run is preserved in `INITIAL_TARGETED.log` (2.573s). Before the final run I strengthened the independently authored missing-field test to retain all other canonical byte ordering and recompute the address from the malformed bytes; this avoids incidental refusal from stale keys or reordered fields. No failing test or weakened assertion was removed.

`CGO_ENABLED=0 go test ./... -count=1`

PASS, all packages, including store 38.615s, resolve 15.110s and daemon 27.886s. Complete output is `FULL_SUITE.log`. This command ran in the independent export, including the final independent tests, rather than relying on the builder's full-suite result.

## Disproof coverage and findings

1. **Complete structured fidelity.** The independent field matrix exercises all 15 string locations: episode, cwd, summary, all declaration text and aliases, all fact text and unparsed ValidFrom, and every Supersedes string. Empty strings where permitted, NUL/newline, non-ASCII, emoji, decomposed accents, U+FFFD, U+2028/U+2029, and HTML-sensitive characters roundtrip. Invalid UTF-8 in every location refuses with nil bytes and empty key. Finite confidence is checked by exact float64 bits for signed zero, smallest subnormal, largest positive/negative finite values and values outside a semantic probability range. No unintended confidence policy is imposed. Exact minimum/maximum int64 nanoseconds, epoch and adjacent nanoseconds roundtrip; one nanosecond outside either limit, zero time and year 1 refuse. Supplied tests additionally verify offset-with-seconds normalization to UTC without caller mutation, ordered duplicate aliases, TypeFallback, nil/empty top-level and alias slices, Supersedes present/absent, NaN/Inf refusal and changed content producing distinct retained addresses.

2. **Complete observation matching.** Twenty-two independent mutations are tested against declaration, source and destination observations. Each mutated original field, all three Supersedes members, pointer nil/empty distinctions, aliases, TypeFallback, confidence, cwd, full episode and exact time participates where that observation contains it. Unaffected observations correctly remain valid: changing episode summary alone, for example, cannot alter an unchanged observation's bytes. Changed observations refuse despite the same ordinal. Negative, immediately out-of-range and maximum ordinal refuse; malformed observations, duplicate members, wrong keys and corrupt revision bytes refuse. Static review confirms decoding rejects a negative ordinal before indexing and reconstructs the full expected canonical observation before comparing exact key AND raw bytes. There is no hash-only comparison or fieldwise subset match.

3. **Strict encoding/corruption.** Independently remove each of 19 top-level/declaration/fact fields, including false/nil fields that typed decoding would otherwise default, while preserving all other JSON byte ordering. Every modified valid-JSON record is addressed by its own actual byte digest and still refuses. Same-value duplicate known members likewise refuse under freshly computed malformed addresses. Supplied tests also cover missing/unknown/case-changed fields, nested duplicates, trailing/leading bytes, lone-surrogate loss, wrong episode and wrong key. Canonical re-encoding and whole-byte comparison are the source of strictness, while reflection checks the typed encoder roundtrip for loss.

4. **Immutable transaction writes and provenance.** Eleven additional raw EP fixtures cover malformed/empty/null objects, trailing junk, missing ID/time, wrong ID/time, duplicate case-insensitive ID, noncanonical escaped ID and invalid UTF-8 in otherwise unrelated bytes. Every fixture makes both list and chunk return their zero result and static refusal. Replaying an already present exact input against bad provenance still refuses. Each swallowed writer refusal poisons the admission owner, prevents finalization and rolls back a separately staged synthetic row. An EP marker staged in the same facade can authorize the input and is removed with it on propagated rollback. Supplied tests cover missing actual EP, occupied different input bytes, duplicate no-op, exact byte preservation of old raw rows, zero graph events, caller alias/pointer changes after staging, and mutation of returned chunk buffers. Two independently decoded objects remain separate even after mutating one and the source raw bytes.

5. **Bounded inspection.** Independently write 107 revisions and prove the 100-key cap, exact ordered first/tail pages and final empty cursor. Invalid count and byte-budget extremes refuse. Existing foreign-episode, truncated, extended, invalid-byte and corrupted exact cursors refuse without returning partial data. Independently assemble a >24KB input through 512-byte complete JSON envelopes while inserting a different revision that removes all declarations/facts after the first chunk; the selected immutable key reconstructs exactly its original bytes. Negative, past-end and maximum offsets refuse. Supplied tests cover budgets 512/1024/24576, exact-end empty completion, byte-limited key pagination, a very long Unicode episode ID, corruption after earlier rows in the same page, and no skip/duplicate. List/chunk entrypoints open one view, with nested full-row and provenance reads sharing that same transaction. Hashed episode keys bound the envelope without truncating the retained ID.

6. **Owner, failures and durable storage.** Independent ordinary AtomicWrite panic removes every staged input row; escaped, nil, zero and closed-root facades refuse. A deliberately ignored validation error in an ordinary scope leaves its unrelated staged write committed, as the expressly preserved ordinary-owner contract requires propagation; this is not an admission-owner bypass. A real Badger value-threshold failure after a successful input stage returns only the static input sentinel, discloses no synthetic private marker, poisons the admission owner when swallowed and rolls back all staged rows. Unchanged supplied tests independently execute actual transaction-capacity errors with allowlisted ErrTxnTooBig, actual commit conflict against a separately committed competing exact key with ErrConflict, propagated ordinary error, admission panic/rollback, reopen, nonempty backup and restore. Full snapshot equality is checked after reopen/restore, including old opaque bytes, and exact input bytes reassemble identically. Physical hardware/disk-fault injection and arbitrary private raw-writer misconduct are not claimed.

7. **New reserved family.** The sole adopter source change is adding `io-input:` to preflight's occupied-family refusal. The supplied tests execute bare, malformed and multi-component occupied keys with empty values in both dry-run and apply; each refuses without candidate writes. There are no production calls to input writer, matcher, list or chunk APIs: a non-test source search finds their definitions and the reserved-family entry only.

## Remaining boundaries

The pure matcher does not prove stored provenance, role/spelling authority, outcome completeness, registration coverage, declaration completion, or that an observation was produced in a selected result. The writer and readers perform their separate EP provenance check. The owner finalizer still needs to bind the actual mutation, dependency, registration, observation and support ledgers. Input summary retention is structured parsed extraction evidence; no source transcript is introduced here.

No current result/head exists in this unit. Semantic Force replay, predecessor/head CAS, exact total ordinal classifications, complete birth replay inventory, branch selection, stale head refusal and lifecycle enforcement across every producer remain future work under the complete design review. Existing ordinary callers must propagate errors; the new writer only adds poisoning where an admission owner actually exists. Private raw transaction corruption, separately created root operations within callbacks, concurrently shared facades and device failure modes are not certified.

The 24KB bound here covers only each internal encoded page/chunk, not a future CLI/RPC wrapper, latency, process memory or complete live concurrent-insertion traversal. Immutable key chunks support stable reassembly; listing is a coherent page snapshot. Adding an adoption refusal prefix alone does not make live adoption or deletion safe. None of the missing integration pieces is relabeled complete by this PASS.

## Exact SHA-256 pins

All relative artifact paths below are rooted in the independent export.

| Artifact | SHA-256 |
| --- | --- |
| INPUT_REVISION_CONTRACT.md | `71e1b29041cd78bc3229674c0396381db2ada44981cddbeede35f6e95779a63d` |
| internal/memory/store/identity_input_revision.go | `b69e6daf91e98fba34166fe949c374d65f9196831d49ddcd11b9650364e59908` |
| internal/memory/store/identity_input_revision_test.go | `14ce28a25f66680c891507ad56fd524f5d1a7d710e3e27e5c1796e6e5d181399` |
| internal/memory/store/identity_input_revision_disk_test.go | `f5ab5f5e6b62d0197a1dbe65fcce4de9af427f52b1dbf7f18d46e4a1a08bcb62` |
| internal/memory/store/identity_legacy_adoption.go | `9ac1ea5299c001e9454c49a54dad0dcb18ed54f04ab9922f04dd02bd7874c3d7` |
| internal/memory/store/identity_input_revision_independent_test.go | `4c670e87d9fa788445ff742d8945bd8bbbdc002e1669d057dcf7783ea06af08c` |
| TARGETED.log | `f8eafefe8a6792d279b61d6b5b35adb7b91f3c4767b58fb5c165db767549830d` |
| INITIAL_TARGETED.log | `3f537105dcbd5a818befc900502f06785141709a74581564686318d14ed56c42` |
| FULL_SUITE.log | `ed85472c5c2641c404b97ea8be14688e6fc93156ea23ef41fe29f7a7b2df57d6` |
| docs/memory-repairs/current-episode-design-independent-review-2026-09-06.md | `a9722fcfad3e21980215dfda2e7ef599ae28216bd4a269dceb616dcf2e5845ce` |
| internal/memory/store/identity_observation.go | `83a39c03435185963a4a2931f94f30f03a7bd1a86c6e4c545ece95580ad9e0ab` |
| internal/memory/store/identity_outcome.go | `7b40e318a2c323efed48a5d40c79853fd61e1455e24265903300e7f3cc7c59e8` |
| internal/memory/store/identity_admission_owner.go | `14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82` |

The report's own SHA-256 is supplied in ARTIFACTS.sha256 and the handoff message. There were no proven counterexamples or preserved failing tests in this review.
