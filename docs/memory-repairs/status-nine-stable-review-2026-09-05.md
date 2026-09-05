# Independent stable-source nine-status live-candidate gate — 2026-09-05

**PASS: the exact nine-entry candidate is ready for the reviewed live retirement workflow on deployed commit `393eeec79f80d3b4becff276c4fcffd71fa68ac5`.** The new source is independently verified quiet, all candidate inputs are freshly reviewed, and the full replica operation, backup restore, raw-state expectations, rollback failures, graph postconditions, and second preview pass. No unresolved violation was proved within this nine-entry scope.

This is a pre-apply gate, not an actual-live result, broad status-cleanup approval, approval of undeployed resolver changes, or final memory-goal completion. The lead still must use the exact reviewed payload, the deployed backup-coupled/transactionally rechecked apply, and independent actual-live postconditions. Any new affected-input drift reopens this gate.

The grader independently extracted commit `393eeec79f80d3b4becff276c4fcffd71fa68ac5` into this directory and ran its own tests. The builder helper and its patched checkout were not used. All writes occurred only in this temporary checkout and its disposable stores. No live writes, deployment, remember call, shared-repository edit, model call, or external research occurred.

## Verified inputs

- Source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-stable-source.badger`, Mini backup `memory-20260905T201914Z.badger`.
- Source size **77,910,575 bytes**, SHA-256 `e994f2021d063082eff59a48cfebc9940975f323551b15b3e11961943409857c`.
- Manifest: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-stable-replica/manifest-replica-only.json`, SHA-256 `a739d9d95de7ac8a8ff8e92ea33e3f717cb1cdf2696d09327c64dbc6bd94f36a`.
- Independent synced/closed backup: `independent-preapply.badger`, **77,910,543 bytes**, SHA-256 `a21569c948db4e312310a1df468215a4071b666dce146b247a57c8057f086b98`.
- Complete raw source and independently restored backup logical hash: `4f892a3d5a3f0e214ebb66d7dd28051012259bf2a90370129bcfb469f02ebaf0`.
- Independently predicted and actual post-apply raw logical hash: `f5ecee171b64edec9d5d7f38bceab78b5c85fba0806b693506b3b0b142678927`.

Logical hashes encode the complete raw key/value map with sorted JSON keys. File SHA-256 hashes are separate measurements.

## Source stability and fresh baseline

The deployed `Store.PendingCounts` returns **0 ready / 0 backoff / 5 parked** on the independently restored backup at `2026-09-05T20:19:15Z`. Exactly five pending records exist, all marked parked. The previously ready `ff7c244…` episode is now committed; no ready queue record remains in this source. `source-queue-truth.json` contains the exact five pending IDs and error states. This closes the specific stability failure proved against the earlier 20:13:33 backup.

Four new episodes were committed since that earlier backup: the cadformats Analytics/Playwright session, a deployment-gate review, a recall-regression attribution investigation, and the ChildScribe repair verification session. Their complete stored summaries and raw graph deltas were inspected. The new source differs through ordinary ingestion in graph and bookkeeping records; this grade derives its baseline from the new source instead of carrying forward old totals.

| Measurement | Stable source | After nine retirements |
|---|---:|---:|
| Entities | 30,457 | 30,448 |
| Facts | 80,242 | 80,242 |
| Historical facts | **7,793** | **7,793** |
| Episodes | 9,325 | 9,325 |
| Current relations | 39 | 39 |
| Cross-type collisions | 482 | 482 |
| Self-loop facts | 1,095 | 1,095 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | **2,855** | **2,855** |
| Logical raw records | 241,498 | 241,489 |

The bold baseline values differ from the older source: ingestion added one historical fact and one hollow entity. They are not effects of this retirement. Full self-loop/dangling/hollow lists and the relation set are identical before and after, not merely equal in count. Existing global defects remain unresolved broader-goal failures.

## Fresh semantic closure review

Every candidate entity, all touching current and historical facts, full source episodes, source-entity metadata, same-episode companions, and all source-touching companion facts were regenerated from this source. All nine still have exactly one current incoming fact, no outgoing fact, and no historical touching fact. Their snapshots and complete independent previews still match the exact manifest expectations.

The new `semantic-evidence.json` is SHA-256 `898a118b79553eb459c3b0f29c7af8506f2bc21acf1f27e32db347d8178ce85e`. It is intentionally not represented as identical to the previous evidence: companion metadata changed. The only differences are:

- Companion entity `jeff` has a repo-reference list rotation: `/Users/jeff/workspace/scribe` falls out and `/Users/jeff/workspace/cadformats` is added. This appears in eight evidence groups. Its name/type/description/aliases/timestamps are unchanged.
- Companion entity `playwright` advances `last_seen` from `2026-09-05T19:12:00.548Z` to `2026-09-05T19:20:42.644Z`, in the publication-result evidence group. All its other metadata is unchanged.

These contextual metadata changes do not turn any measurement into a named identity, change any fact owner, or weaken an artifact veto. A dedicated independent assertion verifies that these two exact metadata edits account for the entire semantic-evidence delta. Every candidate entity, candidate fact, source entity, source episode, and companion fact is unchanged. See `semantic-drift-review.json` for the exact comparison and judgment.

| Candidate | Evidence-backed value role |
|---|---|
| `1-complete-priority-cell` | One-of-24 strict-gate coverage measurement on the retained licensing project. |
| `164-fields-have-field-evidence` | Field-evidence count on the retained named backlog JSON artifact. |
| `193-pages-blocked-on-authority-verification` | Backlog count, despite disputed numerical accuracy. |
| `21-suspect-bodies-removed` | Sanitizer outcome on the retained ingestion engine; distinct from engine/docs/sitemap artifacts. |
| `50-jurisdictions-published` | Seed/publication coverage assertion; not a named release or seed artifact. |
| `641-fields-have-content` | Content count, despite disputed numerical accuracy. |
| `actively-building-maine` | Time-specific builder activity on the existing project. |
| `addressing-review-findings` | Implementer activity, distinct from the retained review thread and rendering task. |
| `ai-sync-tests-15-passing` | Test-result measurement, distinct from the retained program and pytest-file identity. |

The three previously documented factual caveats remain unchanged: the 193-blocked-pages and 641-content-fields assertions conflict with episode `37eb697c…` (1,062 blocked pages / 193 content fields), while episode `53cff505…` reports fifty seeded jurisdictions alongside a companion saying sixteen guides were still unpublished. No factual correction or endorsement is part of this representation-only gate.

## Complete ownership and mechanical checks

The fresh scan covered all **51,774 alias claims**, all 30,457 entities, and all 80,242 facts. Every candidate has exactly its own normalized claim, no hidden extra routing key, no outside entity listing, and no outside match under the actual hygiene punctuation/plural fold. Each candidate has only its one canonical empty reverse index, with no stale, malformed, nonempty, or unreviewed extra adjacency. All existing source identities and `43-url-sitemap` are untouched.

Independent previews are byte-identical to the earlier reviewed previews, SHA-256 `529f6a80140cbce86002bf691d7b86c20395a087c8d0388dd5965a9b7b042114`, and every manifest expectation matches. The preview comparison is supplemented by the semantic closure review above; episode contents and companion facts are not assumed covered solely by manifest fingerprints.

Every replacement is required to equal the complete original fact except for `Dst=""` and `Value=retiredEntity.Name`. Source, relation/raw relation, text, validity, timestamps, confidence, provenance, and any unmodeled JSON field are preserved. No owner is guessed or changed.

The actual raw delta exactly matches 36 deleted records and 27 added records: remove nine entities, nine claims, nine edge facts, and nine reverse indexes; add nine literal attribute facts, nine slug tombstones, and nine spelling tombstones. All unrelated raw records, including all episodes, parked episodes, attempts, cursors, value evidence, and metadata, are identical before/after.

- Ninth-group fingerprint drift aborts all nine groups, preserving the complete raw map and emitting zero events.
- A forced postcondition sees all nine tentative conversions, rejects them, and rolls back every raw change with zero events.
- Successful apply emits exactly 27 events. Its nonempty synced/closed backup restores to exactly the source raw map.
- Every retired exact entity and alias lookup is absent. No new hollow, self-loop, dangling endpoint, fact loss, or historical loss appears.
- A second preview of all nine is nonready/nonapplied and causes no raw write or event.

## Live handoff boundary

This source and exact nine payload have passed the fresh bounded gate. The lead may proceed through the already authorized reviewed live workflow on deployed `393eeec`, while checking that current queue state and complete affected inputs still agree. The operation must retain its nonempty live backup, recheck expectations under its maintenance lock, apply atomically, and receive independent actual-live verification. Newly arrived processing or affected semantic/ownership drift requires rechecking that change; unchanged broad totals alone do not establish closure.

Do not deploy the newer canonical-name patch before applying under this gate and assume it was reviewed here. This report does not grade that code or permit expanding the nine entries, correcting the disputed facts, moving sources, or claiming the broad candidate inventory is complete.

## Reproduction and artifacts

The independently extracted test files and artifacts reside in this directory: `semantic-evidence.json`, `semantic-drift-review.json`, `source-drift.json`, `source-queue-truth.json`, complete before/after entity/fact JSON, `independent-previews.json`, `mechanical-results.json`, `graph-audit.json`, `raw-delta.json`, `raw-payload-and-lookup.json`, `independent-preapply.badger`, `independent-replica`, and `backup-restored`.

```text
go test ./internal/memory/store -run '^TestStatusNineIndependent$' -count=1 -v
go test ./internal/memory/resolve -run '^TestStatusNineIndependentGraph$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusNineIndependentRawPayloadAndLookup$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusNineStableSourceQueueAndDrift$' -count=1 -v
```

All four final probes PASS. The first must be reproduced in a fresh directory because it creates output and backup paths exclusively.
