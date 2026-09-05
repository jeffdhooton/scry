# Independent six-status replica retirement grade — 2026-09-05

**PASS for the exact six-node/seven-fact candidate on the supplied immutable replica source.** The disproof attempt found no named artifact mistakenly retired, guessed source owner, lost assertion/history, alias theft, unexpected raw mutation, or new graph defect. This is strictly replica-only: it grants no current live-apply clearance, no network/crawl authority, and no approval of later deployed code or the broad status inventory.

The source is the previously verified actual-post-nine backup. Live ingestion has since moved beyond it; a new stable source, complete semantic/input recheck, and review against the actually deployed implementation are required before any live six-node proposal. The grader used only independently extracted deployed commit `393eeec79f80d3b4becff276c4fcffd71fa68ac5`, not the builder helper or its patched checkout. All writes occurred in this temporary directory. No live write, model/provider call, remember call, deployment, shared-repository edit, or external research occurred.

## Exact inputs and independently produced evidence

- Source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T202741Z.badger`, **77,914,543 bytes**, SHA-256 `b87def29a4334517299b55e9fc288de141f509b873c4afbdfd79bd101208ae06`.
- Manifest: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-six-replica/manifest-replica-only.json`, SHA-256 **`6b625c190057026d4824e2d89a9146c75697e469e0bb2673d2ffd5e34be7573a`**.
- Independent synced/closed backup: `independent-preapply.badger`, **77,914,543 bytes**, SHA-256 `3951b18dc107b93c8257057fbd49f704596558e3db73348798446d3f628fa441`.
- Complete source and independently restored backup raw logical hash: `f5ecee171b64edec9d5d7f38bceab78b5c85fba0806b693506b3b0b142678927`.
- Independently predicted and actual replica post-state raw logical hash: `d9b6adbd13df7234ece9292e53777d3810f100abaeada9aa5383f704b7810408`.
- Full semantic evidence: `semantic-evidence.json`, SHA-256 `b4349072a582c1ee999e7084b89a78f8d87d8a1e440e93043baa9bb54c027bcb`.
- Complete 82 same-episode companion facts: `all-same-episode-companions.json`, SHA-256 `250df05c75f5634719c128dbdcd25aefa0d0bcd2a3d3837327c7f4c5a710b59f`.
- Independent previews: `previews.json`, SHA-256 `e92dfb14b34de01cae3ec79cdfd4ec675eee0f377bc4992de4e601a48ab241c1`.

Logical hashes cover sorted JSON maps of complete raw Badger keys/values; physical backup hashes are separately measured.

## Semantic disproof

The grader read every candidate's full metadata, every current/historical touching fact, every complete source episode, the complete 82-fact union of same-episode companions, and relevant source/artifact metadata. All source-touching facts are also retained in the evidence artifact. Every selected touching fact is current and incoming; none of these six nodes has an outgoing or historical touching fact. All six have no listed aliases and exactly one owned normalized claim.

| Candidate | Incoming facts | Evidence-backed decision |
|---|---:|---|
| `app-tests-32-passing` | 1 | A numerical result from the earlier two-release update. Its description and fact distinguish that stage from the later discovery result; this is not the test-suite identity. |
| `app-tests-34-passing` | 1 | A later numerical result after discovery machinery landed. The same-episode `all-gates-passing` companion explicitly lists 34 app/18 ingestion tests at commit 0012ca7. |
| `canonical-route-sitemap-passing` | 1 | Manual episode `7ce24fe4…` explicitly reports successful canonical-route/sitemap checks after deployment 52489a6. The project, guide, migration, and route/sitemap artifacts are retained. |
| `indexing-audit-clean` | 2 | Metadata says audit outcome, and manual episode `e9a22440…` reports no indexing leak/malformed page. Both the site-level result and the sitemap-level 42-URL assertion survive under their existing sources. No evidence makes this label a named audit artifact or durable invariant. |
| `live-checks-passing` | 1 | Manual episode `d0c2c19d…` reports routes/sitemap passing after deployment adb2c25. It is the result state, distinct from the actual deployment and migration. |
| `network-authorization-pending` | 1 | Episode `d0963022…` says the prepared crawl cannot run without explicit authorization. The label is the withheld-permission state; the prepared plan, runner, and authorization safeguards remain distinct retained records. |

The 32/34 assertions have the same stored day-level `ValidFrom`, but different values, text, and confidence. Their earlier/later distinction comes from the stored descriptions and fact wording, not invented timestamp precision. Both complete assertions remain separate attribute keys; neither replaces, invalidates, merges, or overwrites the other.

`indexing-audit-clean` becomes exact literal **`indexing audit clean`** on both `sitemap` and `statelicenselookup`, preserving their distinct full fact texts. The existing `sitemap` entity lists repo references for multiple projects and a description concerning other sitemap content. This is an existing identity-scope problem, not permission to infer a new project-specific owner. Both existing sources, all source metadata, and all unrelated facts remain unchanged. This grade does not endorse global sitemap ownership.

For `network-authorization-pending`, the relation remains **`blocked_by`**, raw relation stays absent, and the entire requirement for explicit network authorization remains verbatim. The approved conversion provides no authorization to fetch or crawl anything. Same-episode companions continue to require exact plan hashes and explicit batch approval.

`gate-green` is excluded. In this particular source it has 20 current touching facts, five outgoing facts, and zero historical touching facts; its alias set and unresolved source-owner questions are not reviewed for retirement here. It and all its facts/claims remain byte-for-byte unchanged. The explicitly protected `43-url-sitemap` and all other noncandidate artifacts also remain unchanged.

## Complete mechanical proof

The independent scan covers all **80,242 facts**, **30,448 entities**, and **51,765 alias claims**. Each candidate's complete relevant claim set is owned by itself, with no extra hidden routing key, outside normalized listing, or outside match under the actual hygiene fold. There are exactly seven empty canonical reverse indexes and no stale, malformed, nonempty, or unreviewed candidate adjacency. All manifest entity/fact/alias/adjacency expectations match independently generated ready previews.

Every replacement must equal the complete original fact except `Dst=""` and `Value=retiredEntity.Name`. A generic JSON-object comparison additionally prevents hidden/unmodeled fact fields from being dropped. Source, relation, raw relation, text, time, validity, confidence, and episode provenance remain identical.

The independently predicted whole-store delta is exactly **26 deleted records and 19 added records**: six entities, six claims, seven old edge facts, and seven reverse indexes removed; seven attribute facts, six slug tombstones, and six spelling tombstones added. The complete actual raw map matches that prediction. No unrelated record changes, including episodes, queue, attempts, cursors, prior nine-retirement markers, and metadata.

| Measurement | Before | After |
|---|---:|---:|
| Entities | 30,448 | 30,442 |
| Facts | 80,242 | 80,242 |
| Historical facts | 7,793 | 7,793 |
| Episodes | 9,325 | 9,325 |
| Current relations | 39 | 39 |
| Cross-type collisions | 482 | 482 |
| Self-loop facts | 1,095 | 1,095 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | 2,855 | 2,855 |
| Logical raw keys | 241,489 | 241,482 |

Complete defect lists and the relation set are identical, not just their counts. These preexisting global defects remain outside this bounded PASS.

## Atomicity, restore, lookup, and second preview

- A wrong sixth-group fingerprint aborts the whole batch with identical complete raw state and zero observer events.
- A deliberately failing postcondition sees all six retirements and all seven replacements inside the transaction, then rolls back every mutation with zero events.
- The real replica apply uses the deployed backup-coupled primitive. Its nonempty synced/closed backup independently restores to the full original raw map; successful apply emits exactly 20 expected fact/entity events.
- All six retired exact entity/name lookups are absent. All seven values are found under their original sources. `sitemap` and `statelicenselookup` both resolve to their own source slugs; both indexing values are retrievable. The 32- and 34-test values are simultaneously present.
- The second exact six-entry preview is nonready/nonapplied and changes no raw record or event.
- An additional exact CLI dry run, independently built from the deployed archive and pointed at the restored replica, returns `dry_run=true`, `applied=0`, `refused=6`, with each retired entity reported absent. A subsequent whole-store raw hash check proves the CLI changed nothing.

## Scope and reproducibility

This candidate passes only on the named immutable source and deployed commit. Live graph growth and queued work reported by the lead make this source unsuitable as an immediate live fingerprint authority. After any deployment and queue stabilization, restore a fresh source, regenerate exact expectations, recheck full source episodes/companions and outside claims/folds, and obtain the corresponding fresh gate. Do not merely refresh hashes or extend this approval to another node.

Artifacts reside in this directory: complete semantic evidence and companions, full before/after entity/fact exports, previews, `mechanical-results.json`, `graph-audit.json`, `raw-delta.json`, `source-values.json`, `exact-cli-second-preview.json`, `cli-preview-no-write.json`, the independent backup/restore, and grader-written tests.

```text
go test ./internal/memory/store -run '^TestStatusSixIndependent$' -count=1 -v
go test ./internal/memory/resolve -run '^TestStatusSixIndependentGraph$' -count=1 -v
go run ./cmd/scry memory retire-entities --file /tmp/scry-migration0160-fresh-sep05.teYtyC/status-six-replica/manifest-replica-only.json --dir /tmp/scry-status-six-grade.rnpwPR/replica
go test ./internal/memory/store -run '^TestStatusSixExactPreviewNoWrite$' -count=1 -v
```

All final probes PASS. The main test creates output/backup paths exclusively and must be reproduced in a new temporary directory. No benchmark, sweep, current live-state readiness, or broader inventory-completeness claim is made.
