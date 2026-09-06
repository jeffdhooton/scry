# Independent second real postguard sweep extension

Verdict: bounded PASS for the independently restored second snapshot, complete old-assertion/provenance preservation, unchanged structural defect sets under the proper Name+Aliases contract, observed preserved-input fact-conflict refusal, completed durable note, and unchanged five-suite offline recall controls. This is not a whole-source or final-goal PASS. The second sweep scanned real files but found no new episodes; later successful extraction processed already queued input. The first-sweep proof remains separately frozen in `FIRST-SWEEP-REVIEW.md`.

All work remains private under `/tmp/scry-first-sweep-independent.flj8Ff`, using the independently archived exact production commit and inspected/extended helper described in the first report. No source transcripts, provider calls, queue retries, shared/live writes, credential actions, repairs, deployment, notes or room messages were performed. Actual graph/source identifiers and previews remain hash-only.

## Direct restore and distinct real sweep

I completely read and independently verified `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/memory-20260906T010908Z.badger`: 74,478,385 bytes, SHA-256 `905421a42701e176d13ad17526e31215c38e7cd485b9d4283923c313deed8ed6`. Direct Badger Load into a new private directory precedes candidate Store.Open. Complete raw maps before and after Open, all facts/entities, full offline index, actual recall and Close are equal. The read payload was 11,457 bytes.

The raw map has 244,833 records and digest `aac22ac2914b1bddbd516aaca97a7dc2bfe8abe2246d8481f0a2c697438ff90a`. It contains 81,146 facts (73,299 current / 7,847 historical), 30,884 entities, 9,395 episodes, 52,313 alias claims, 55,975 adjacency entries, 11,031 attestations, 3,149 cursors, 879 value-evidence records, five metadata records, fourteen pending records and the same 42 rejection/retirement markers.

Persisted last_sweep advances from the first real sweep's 01:01:21.255638 to 01:04:14.853735 UTC. The second stored report records 94 files scanned, zero files ingested, zero new episodes and zero errors. Its report host digest differs from the preceding report; I make no unverified machine-attribution claim. This is a distinct completed scan, not the 00:49:06 process startup. Last successful extraction then advances to 01:04:38.934663 UTC. Thus two real sweep timestamps are observed after the reviewed deployment, but the second scan itself did not discover new extraction input.

## Complete first-to-second and base-to-second preservation

| Family | Added | Changed | Removed |
|---|---:|---:|---:|
| adj | 26 | 0 | 0 |
| al | 28 | 0 | 0 |
| ar | 0 | 0 | 0 |
| att | 21 | 2 | 0 |
| cur | 0 | 0 | 0 |
| en | 21 | 10 | 0 |
| ep | 3 | 0 | 0 |
| fa | 40 | 1 | 0 |
| meta | 0 | 3 | 0 |
| pq | 0 | 1 | 3 |
| rs | 0 | 0 | 0 |
| rt | 0 | 0 | 0 |
| ve | 3 | 3 | 0 |

Every old fact assertion, confidence and provenance is retained. All 7,847 first-snapshot historical facts are exact. The sole changed fact appends episode provenance only: key SHA `db992b769eca88b9ee10e2f706cbfb281ca25188b43126bf83c8489b8f889f91`, prior payload SHA `215cc1ffc39a27ce9437ed3b77a3299477cf6f832460b74e421d208c0c184747`, later SHA `4a1250d2e2816e5431c3f3c09942a7ab9d93d1cf6b589bb2c7a7eebe938cad0d`. No fact key is removed and no old text/value/endpoints/raw relation/validity start is replaced. Every one of the 40 added facts has nonempty source provenance, cites a newly stored episode, has complete episode closure and existing endpoints.

All 42 protected markers and thirteen previously parked queue payloads remain exact. No old alias claim changes owner or disappears. All old episodes and cursors are exact. The two changed attestations retain every old episode ID and complete episode closure. Three changed value-evidence records retain their normalized spelling, all old spellings and all old episode IDs, with complete episode closure.

The ten old entity changes involve only last_seen, aliases and repo_refs. No old name/type/description or alias is removed. One six-entry repo_refs list adds one reference and evicts an old reference, exactly reproduced by the unchanged six-reference AddRepoRef behavior. The entity-key SHA is `c9dc52065f28dbc92de76ea0cb5740c780e7b83f375b0f6ee1faa26009ae9ee8`. The complete hashed reference difference is retained in `second-details.json`. This metadata eviction is explicitly reported; no claim of preserving every old repository reference is made.

I also computed a direct complete 00:54:30 base-to-second delta, independently of composing the adjacent summaries. All 81,057 original assertions and all 7,845 original historical raw records remain preserved; there are 89 added facts and exactly two changed old facts, comprising the first interval's preserved invalidation and the second interval's provenance append. No original raw key disappears in any family. All original 42 markers, thirteen parked payloads, 52,258 claims and 9,388 episodes are exact. The direct full delta records all keys/payload hashes and establishes the final raw digest above.

## Three completed inputs and one preserved fact-conflict refusal

All three removed pending rows have exactly matching new stored episode IDs. For each, source, source reference, occurrence time, working directory and repository attestation equal the pending input. The three completed sources are one Claude, one Codex and one manual. No queue removal is unexplained.

The root's once-submitted note is the manual completion. At 01:03:09 it was pending at attempts zero; at 01:09:08 it is ingested and absent from pending. Raw completed episode SHA is `9096d5e5df20a1b5e64b313d5de0bd2761549a8c31ac687ebc8c35224808c337`, matching the root's independent receipt. Its prior queue input/source metadata compare exactly. This demonstrates durable completion of that submission, without a reviewer retry. The supplied 97 ms acknowledgment remains one supplied observation, not a measured p95.

The fourth active pending record becomes the fourteenth parked item. Pending-key SHA is `42e7eff786011d30cef0918c21495e6e032ae2f59ac79f221c40324d857d88b8`. Only attempts, last_error and parked change: attempts 0 to 1 and parked false to true. Every other field, including full text, original source metadata, hints, force, enqueue and next-attempt time, is identical. Prior payload SHA is `ce8adb2d5bb158193d59ae0f8bef629aa78d1572b49902fcae6ae624778e50f5`; later SHA is `dbee62e6ff6eca9318ff16232008ba6ec0e398250bdd4131ac656249eef3b05e`. Error SHA is `1d2e95890dab9a1d96d539b34564782bbf6f12698b2276f23b476764212c3bb9`.

In-memory classification using exact production sentinel strings gives ErrFactConflict=true, ErrAliasClaimed=false and ErrAliasRejected=false. The error text itself is never emitted. The failed episode is not committed, and zero stored facts cite its ID.

The error's occupied-key digest is `0e2213a6a90f9a3beb31f65b5db490c97b5c12e6c113cfd284d3ac3bc18537bd`. No fact key with that digest exists in either compared snapshot. Therefore this particular refusal does not prove preservation of a preexisting persisted occupied key. It is consistent with two conflicting assertions occupying a newly staged slot within the subsequently rolled-back episode, but the source/extraction output was not replayed, so that mechanism is an inference, not a reproduced semantic diagnosis. The complete delta independently proves old persisted assertions survive. The new guard has now demonstrably refused a live input and preserved that input for review; it has not made a semantic repair decision.

## Structural sets, exact lookups and remaining semantics

All proper Name+Aliases sets are byte-for-byte equal across first and second snapshots: zero missing claims, 504 wrong-owner listing occurrences, 29 claims unlisted by the indexed owner, 462 multiple-owner normalized listings, 27 multi-type listings and zero dangling alias owners. All these sets also equal the base through the independently measured first interval. Existing unexplained claims and collisions remain unresolved.

The broad Slug+Name+Aliases inventory gains four additional missing-index rows, 3,881 to 3,885. All four are slug-only, as are all 3,885 broad missing rows. None is a listed canonical name or alias; proper missing claims remain zero. The prior report's correction is retained: all 3,881 first-snapshot broad missing rows were already slug-only. Storage slug fallback makes these broad entries unsuitable as evidence of stale aliases. All 21 new entities pass direct slug lookup; all 26 actual names/aliases resolve to the correct owner. The whole raw map remains unchanged afterward.

Every other complete structural set is unchanged: 2,441 dangling endpoint occurrences, 2,851 entities with no touching facts, 2,969 with no current touching facts, 1,099 self-loop keys including 93 current, and zero missing provenance episodes or missing/extra adjacency. Both broad and proper inventories retain full set digests and explicit added/removed hash sets. Exactly 39 documented relations remain current, with zero noncanonical current facts.

No new hollow entity, dangling endpoint, listed-name/alias defect, cross-type normalized listing or self-loop is demonstrated in either interval. This does not prove that the 46 combined new entities are semantically valid identities rather than contextual values. No name heuristic, provider probe or transcript reconstruction was used to grant a status-value PASS. A complete contextual new-identity/status audit and production hygiene-proposal review remain open.

## Five unchanged offline suites at second snapshot

Five additional independent candidate CLI commands use the unchanged archived suite files and top=20 against the frozen second replica. Every command completes, preserves the full raw map and returns zero cap violations. Hits and mean ranks match both base and first snapshot: heldout-2026-09-03 51/62 and 4.8431372549019605; heldout-b 29/66 and 5.068965517241379; probes 7/7 and 1; tuning-strict 45/50 and 4.511111111111111; tuning 47/50 and 3.978723404255319. Maximum payloads respectively are 12,101, 13,358, 9,961, 11,530 and 11,530 bytes. All remain below 24,576 bytes. Full ordered missed-question/rank lists are compared in the safe benchmark comparison artifact.

These are graph-drift controls with the deployed binary, not new questions or restored original suite floors. The first two original floors still fail. Full successful-query rank/payload equality is not inferred from aggregates. No fresh fifty-question grader was run.

## Scope and reproduction

Two distinct actual scans and successful postdeployment queue processing are now evidenced. That is insufficient to certify the goal's final two-sweep clause: cleanup and semantic classification are incomplete, preexisting structural defects remain, no complete hygiene no-op is proved, history/credential blockers remain, and the original benchmark floors and other final gates are open. No final source safety, identity repair approval, credential remedy, remember p95, coverage/orient regression verdict or two complete fresh grading rounds follow from this bounded PASS.

Reproduce with private helper `restore SOURCE second SHA256`, `delta after second`, `delta before second`, `sweep-audit after second`, `sweep-details after second`, `sweep-closure after second`, `proper-claims after second`, `queue-transition after second`, `conflict-closure after second`, and `bench-one second ROOT /Users/jeff/go/bin/scry`. Full source, hash-only evidence and both reports are pinned by `ARTIFACTS.sha256`.
