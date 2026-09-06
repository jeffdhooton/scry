# Immutable outcome codec independent review — initial candidate

2026-09-06. **FAIL / NO-GO on pinned implementation `fc7f928b`: an actual storage refusal discloses original outcome bytes in its returned error.** This is one reproducible contract violation demonstrated on both memory-backed and disk-backed Badger. The other bounded preservation checks passed. This report does not grade a corrected candidate.

## Scope and isolation

I ran `scry memory orient --cwd .`, read the complete active goal, complete archived controller V3 independent review, full OUTCOME_CONTRACT.md, full outcome implementation and builder tests, and the relevant observation, generation and AtomicWrite implementation. I exported baseline `fd464c22fd8db22d31edce8bf09457ec2d10f726` with git archive into `/tmp/scry-outcome-review-fV5kP0`, then copied the three pinned candidate files with apply_patch. I authored only independent tests, this report and captured test output in this private export. No root/shared/frozen source edits, live graph writes/reads beyond mandated orientation, provider calls, sweeps, configuration changes, memory writes, room posts, deploys or external writes occurred. Source tests use synthetic local Badger databases.

This is the private, uncalled preservation codec review requested by V3. No controller integration, identity authority, support classification, endpoint dependency closure, legacy adoption, Force current projection, cleanup, recall/live performance, or goal-completion clause is certified.

## Reproduced violation

`internal/memory/store/identity_outcome.go:260–261` calls `st.txn.Set` and returns the raw error. Badger v4.9.1 `txn.go` `modify` calls `exceedsSize` on values above a configured storage threshold; `exceedsSize` includes a hex and ASCII dump of the first 1,024 value bytes. The outcome's first bytes include the episode ID, complete birth tuple, original primary name and Supersedes spellings. This directly violates the contract sentence “Errors reveal only digests and static reasons.”

Independent `TestIndependentOutcomeStagingErrorDoesNotDiscloseInput` opens real Badger in memory mode with a 4,096-byte value threshold. Its valid observation and actual episode commit first. A valid outcome containing opaque JSON materialization then reaches the real Set call and refuses at 35,853 bytes. The returned error starts `Value with size 35853 exceeded 4096 limit. Value:` followed by the canonical outcome dump. The test checks snapshot equality and zero observer events before failing its privacy assertion. It fails in both targeted and full runs.

Independent `TestIndependentOutcomeDiskStagingErrorDoesNotDiscloseInput` separately opens real disk-backed Badger with a 1,048,576-byte value-log limit. Its valid 1,561,189-byte canonical outcome reaches Set and refuses with the same dump, starting `Value with size 1561189 exceeded 1048576 limit. Value:`. Snapshot equality and zero observer events pass here too. This rules out the finding being restricted to memory-only storage or a mocked error.

Both failed safety tests remain unchanged after their first run. The original independent suite file was not edited when the disk reproduction was added as a separate file. Production code and the copied builder tests remain byte-identical to the supplied pins. The original builder fixture correction from 190,000 to 120,000 materialization bytes is recorded by the root; this reviewer received only the corrected test hash shown below and did not recreate or amend its earlier failure.

Smallest correction: sanitize every error returned from Set to a static outcome error. If callers need a specific failure classification, preserve only an explicitly allowlisted static sentinel, such as `badger.ErrTxnTooBig`; do not wrap arbitrary Badger errors whose Error method retains raw bytes. Exhaustive inspection of this codec's other error exits found no second raw-storage error route: generation reads and provenance failures are replaced with static errors, and outcomeView sanitizes reader errors. This is a code inspection finding, not proof of every future dependency path.

## Other executed bounded evidence

The independent suite has six top-level tests including the two failing staging-disclosure tests. Its four other top-level tests passed, including fourteen stored-corruption/provenance subcases:

- A primary unresolved assertion with both original Supersedes fields has no invented birth or generation. The copied builder's hint-only case independently executes as well. Exact spelling includes whitespace, punctuation and Unicode. Two registered birth outcomes reference the two original sides of one fact without creating a graph entity or fact. Explicit resolved annotations remain descriptive, including an arbitrary syntactically valid owner slug that receives no alias routing.
- Identical replay keeps the same key. Changed extraction text at the same episode/ordinal produces a separate observation revision and outcome; linking that changed revision to the old predecessor refuses. Same-input successor and branch records remain distinct and inspectable. Existing raw observation bytes remain unchanged.
- Twenty-seven oversized outcomes paginate through full keys at budgets 512, 777 and 24,576 with limit 100. Comparing the complete sequence to sorted expected keys finds no skip/duplicate. Chunk JSON is actually marshaled and unmarshaled, its complete envelope stays within each budget, offsets advance exactly, and assembled bytes equal the stored canonical value. Disk close/reopen and nonempty backup/restore preserve the complete raw key/value snapshot.
- Missing episode, wrong episode time, duplicate case-folded episode ID, missing observation, unknown observation member, occupied wrong outcome value, unknown/duplicate outcome member, invalid UTF-8, unpaired surrogate, wrong role/spelling/ordinal and corrupted predecessor all refuse on detail and list. Returned results are zero values rather than partial data. Reader calls preserve the complete snapshot.
- Root, nil and expired facades refuse writes. Callback error and panic preserve raw snapshots. Mutating the original ordinal pointer and link slice after staging cannot change canonical bytes. Nil materialization remains distinct from empty/non-object/invalid bytes on assertion outcomes.

The copied six builder tests also pass, covering fifteen mutation refusals, three rollback cases including real ErrTxnTooBig, declaration birth normalization, opaque unknown/duplicate materialization fields, absent materialization, birth/materialization caller mutation, exact raw backup equality, invalid reader budgets/cursors/offsets, and zero graph routing/events. The full repository test results exercise normal routing/recall regressions in local fixtures. They are not live benchmark grades.

The codec permits descriptive “supported” records without proving live staging and permits committed annotations marked not-materialized. These are not findings against this contract: it explicitly disclaims support certification and semantic disposition integration. A predecessor certifies only its direct input lineage; ancestry/head/branch selection is deliberately excluded. Bounded pages are snapshot reads and do not certify concurrent insertion traversal.

## Commands and results

All commands ran in `/tmp/scry-outcome-review-fV5kP0` with CGO disabled.

1. `CGO_ENABLED=0 go test ./... -count=1` before adding reviewer tests: PASS, including store 25.023s and daemon 28.095s.
2. `CGO_ENABLED=0 go test ./internal/memory/store -run TestIndependentOutcome -count=1 -v` with initial reviewer file: FAIL only staging-disclosure test; other four reviewer top-level tests and fourteen corruption subcases PASS.
3. `CGO_ENABLED=0 go test ./... -count=1` with initial reviewer file: FAIL only staging-disclosure test; all other packages pass.
4. `CGO_ENABLED=0 go test ./internal/memory/store -run TestIndependentOutcomeDiskStaging -count=1 -v`: FAIL, same disclosure reproduced on real disk-backed Badger.
5. Final `CGO_ENABLED=0 go test ./... -count=1` with both unchanged reviewer files: FAIL solely the two staging-disclosure demonstrations; store 26.029s, daemon 28.221s. Complete output is `INDEPENDENT_FULL_TEST.log`.

No test was weakened or removed to produce a passing result. This initial candidate remains NO-GO pending a separately reviewed correction with both failed tests retained.

## Exact pins

| Artifact | SHA-256 |
| --- | --- |
| Complete active objective | `758153967e3a91f0cf6168bf9305e9c4e1be259e7fb552d8df32a64555fb2463` |
| Archived controller V3 independent review | `2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10` |
| OUTCOME_CONTRACT.md | `e5bd83f2c2e30355fe8fa1642420305c51299ccb40d847853ed3ea619a0f7ab2` |
| internal/memory/store/identity_outcome.go | `fc7f928bd056db3787a92f6a9997262435e17547330cbbfbc83ba063094a31df` |
| internal/memory/store/identity_outcome_test.go | `25407a0dedf43919936b27c99e4709fd4ab2dec6dc745007b783e917317df419` |
| internal/memory/store/identity_outcome_independent_test.go | `4d122a3de3a72e7032492d30f15c5e70330c41f47501873554f1cfd6e0884a4f` |
| internal/memory/store/identity_outcome_disk_disproof_test.go | `2528870d02209a6eb2c032fc820a6137c42afaa061a38c2e5ae11fd9ae72cf2a` |

Relative paths identify the private export. Report and complete-log hashes are supplied separately to avoid self-hash drift.
