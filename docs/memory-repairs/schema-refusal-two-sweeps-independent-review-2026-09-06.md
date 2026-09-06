# Independent schema-refusal later-ingestion and two-sweep review

2026-09-06 UTC. Private evidence directory: `/tmp/scry-schema-two-sweeps.AZXYyV`.

Verdict: **FAIL for final graph quality**. One new entity with no current or historical facts appears after deployment and remains through both explicit sweep snapshots. The original held-out recall floors also remain unmet. This is not a whole-goal grading round. Within the narrower preservation scope, all original assertion content, validity, confidence and provenance survives; the persisted reports independently establish two subsequent sweep completions. Their start chronology is attributed to the lead's tool receipts, not independently proven from persisted metadata.

## Independence and method

I ran `scry memory orient --cwd .`, read the complete active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, and read `docs/memory-repairs/schema-refusal-actual-independent-review-2026-09-06.md` completely. Its independently rechecked SHA-256 is `c3fde6b6071652ddeaf33e76794db8ab21527bf24feffe5f25873cb3c0afa4ba`. That prior report covers the immediate deployment interval, not the later intervals graded here. Lead-agent counts were treated as hypotheses; all counts below came from my own fresh direct restores and inspections.

I exported `a078240` with `git archive a078240 | tar -x -C /tmp/scry-schema-two-sweeps.AZXYyV`. I read the complete source of the inventory, structure, closure, queue, benchmark, rank and drift helpers from `/tmp/scry-first-sweep-independent.flj8Ff/cmd/actual-grade` and `/tmp/scry-schema-actual-disproof.MJiRYo/source/cmd/independent-{evidence,drift}`, copied them into my private export, and rebuilt them locally. Private changes used apply_patch. Changes hash unexpected errors, use Name+Aliases for the full structural audit, exclude all four retry fields when comparing queue inputs, and add independent persisted-sweep, note, new-parking, changed-spelling lookup and empty-entity attribution inspections.

No stored fact text, value, key, value-derived slug, pending text, source reference or actual error was printed. Full raw maps remain in memory for comparisons; output contains hashes, field names, counts, timestamps and allowlisted entity types. Benchmark fact previews were hashed in memory before output. All restores use direct Badger Load into nonexistent private destinations, never populated application Restore. No live/shared writes, provider calls, sweep invocations, queue retries, deploys, restarts, config changes or shared-checkout edits were performed by this reviewer.

## Artifact and complete backup verification

Both installed host binaries were independently checked at their exact paths: `/Users/jeff/go/bin/scry` and `/Users/jclaw/.local/bin/scry`. Both SHA-256 values are `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`. Their corresponding `.pre-a078240-20260906T0146Z` binaries both hash to `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`. My own independent build reproduced the installed hash exactly:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=a078240' -o /tmp/scry-schema-two-sweeps.AZXYyV/scry ./cmd/scry
```

Backup sources are `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T<TIME>Z.badger`. Each whole file was hashed before direct load; load completed, and every active logical key/value byte remained equal before and after application Open, AllFacts, Entities, index construction and bounded local recall.

| Time UTC | Backup bytes | Backup SHA-256 | Logical rows | Facts/current/historical | Entities | Episodes |
|---|---:|---|---:|---|---:|---:|
| 02:03:46 | 74,921,236 | `37187891a0bb081db2f35754df726432caba010e77fba5896c166c2ffa1b866e` | 245,425 | 81,304 / 73,449 / 7,855 | 30,956 | 9,409 |
| 02:09:13 | 75,046,526 | `6c762c49301660ff2eeb07639a820c2f6bc99a3680964cfbccd6a210e606e6d7` | 245,686 | 81,377 / 73,519 / 7,858 | 30,994 | 9,415 |
| 02:14:53 | 75,163,942 | `a3f64e3c32b665f82604f6dc9f24651dab915290c40cffc2cf8b1fd8edccf991` | 245,690 | 81,377 / 73,519 / 7,858 | 30,994 | 9,415 |
| 02:17:07 | 75,203,524 | `6db34cef1e9128e10cb1023200f85b25ce4af3331e0fe26709a20d7b427d2d2f` | 245,718 | 81,384 / 73,526 / 7,858 | 30,997 | 9,416 |

Logical digests, length-framing every key and value in iterator order, are respectively `022b8739f013ed98e2b9e96df7a117959fa533335bb43f5b32bfbd047a593f28`, `4d5e9f62c649684203c0abfdc4e8e27c19ae9dcced35154b9dbf38b5d79cc1e4`, `12ba496327c7b5accd12d6b2b3597edefc42ea0925af6569a19884aaa413379e`, and `eebab8549217f5544450dc0e1ec574ad47c9fa6317c60213a48c68a6e8f26f6d`.

## Full raw and assertion accounting

All three adjacent intervals and the full 02:03:46→02:17:07 interval were compared. Every observed family was enumerated; no unknown family was omitted. The complete full-interval accounting is:

| Family | Old rows byte-equal | Added | Changed | Removed |
|---|---:|---:|---:|---:|
| reverse adjacency | 56,091 | 58 | 0 | 0 |
| alias claims | 52,428 | 50 | 0 | 0 |
| alias-rejection markers | 4 | 0 | 0 | 0 |
| attestations | 11,065 | 30 | 1 | 0 |
| scanner cursors | 3,154 | 5 | 2 | 0 |
| entities | 30,942 | 41 | 14 | 0 |
| episodes | 9,409 | 7 | 0 | 0 |
| facts | 81,300 | 80 | 4 | 0 |
| metadata | 1 | 0 | 4 | 0 |
| pending | 15 | 3 | 10 | 1 |
| retirement spellings | 19 | 0 | 0 | 0 |
| retirement targets | 19 | 0 | 0 | 0 |
| value evidence | 930 | 20 | 12 | 0 |

All 81,304 original facts remain at their exact original keys. Four current facts gain episode IDs only, with every original episode retained; assertion text, endpoints, value, raw relation, valid_from, invalid_at and confidence are equal. The remaining 81,300 full fact payloads are byte-identical, including all 7,855 original historical records. There are zero fact removals, relocations, backdates, invalidations or changed historical validity timestamps in this interval. The three additional historical facts were newly added records, not invalidations of old assertions. All 42 repair markers and 52,428 old alias claims remain byte-identical. This distinguishes real fact additions from exact-key preservation; nothing is counted as a relocation or a no-op merely to reconcile totals.

All 80 added facts have existing endpoints and complete episode provenance; each cites at least one episode new to the initial snapshot. The one changed attestation retains all earlier episode IDs and resolves every new ID to a stored episode. Twelve changed value-evidence records retain their normalized identity, every old spelling and every old episode; they change only episode lists and, on four records, spelling lists. Closure checks pass.

Fourteen preexisting entities change metadata. Core slug, name, type, description and created_at remain equal. All fourteen update last_seen; six also change repository references, and two also add aliases. Every old alias is retained. One repository-reference list loses an entry during 02:14:53→02:17:07: entity key SHA-256 `ec63f8c772a30cb03751324403d426508ab2b78541622c8e70b443b1687c06cf`, six refs before and six after, removed reference hash `906ea6f17856346d3de8b9a34d767bde23edb150c7ce7cbc88e53f0a8d605696`, added reference hash `7dc66df5ed87909268b360a5d75564dc72e6054d74806d2bbb241f73da048c85`. Calling the existing pure `resolve.AddRepoRef` on that old list and actual added ref reproduces the final list exactly. The source enforces the newest six refs; this explains the mutation without certifying it as an approved semantic loss. The complete metadata delta is not wholly additive. Older predeployment backdating and other capped-list losses remain outside this interval and are not repaired by this finding.

## Persisted sweep and queue evidence

The earlier automatic report completed at 02:04:26.083309 UTC with 2,388 files scanned, 8 files ingested and 14 episodes. It is excluded from the requested explicit pair because its start may precede deployment.

| Backup containing report | Persisted finished_at and matching timestamp UTC | Files scanned | Files ingested | Episodes | Errors |
|---|---|---:|---:|---:|---:|
| 02:14:53 | 02:14:01.927205 | 2,390 | 2 | 2 | 0 |
| 02:17:07 | 02:16:31.642506 | 2,390 | 1 | 1 | 0 |

The report payload hashes are `f340bd4816cc40f9f9a3c5dec4683e78c9208cd5f56e9649e3d35870866b5d35` and `9bd89fdaad20620d9b67c5333326ea34fa589e173e7641faafac402ff2acb1c8`. Both have host-field hash `81a40dae388c60e69edd4ba20a02b3b22d6d4e3d772620b11bea77b0fc986a2c`; this review does not map that hash to a named machine. Source keys are separately hashed in the evidence, and no hostname inference is used. The report/stamp pairs were read directly from each complete restore. The daemon overwrites finished_at with its current time when it stores the report; the sweep code calls the report path only on non-dry-run completion. These fields support actual sweep completions but do not store a start time, invocation binary hash, extracted-fact count, or successful queue-drain guarantee.

The lead reports two explicit installed-laptop `memory sweep` CLI invocations, the first launched after its 02:12:14 UTC clock observation and the second after the 02:14:53 backup. Those command-start details exist in the lead's tool receipts; I did not independently inspect a timestamped invocation artifact. Completion times are independently proven and are after the reported 02:03:32 deployment. No claim is made that the persisted records alone independently prove the CLI start chronology. A sweep's episodes count reflects distillation/enqueueing, not completion of provider extraction; the graph changed by six episodes before the first explicit snapshot and one episode between the pair.

The 15 originally parked pending records remain fully byte-identical. Five formerly active records become parked, two before 02:09:13 and three before 02:14:53, leaving 20 parked records. Each new parking matches the fixed alias-claimed error predicate; none matches fact-conflict or alias-rejected. Every complete non-retry input is retained, excluding only attempts, last_error, next_attempt and parked. All five have no committed episode and no fact citing their episode ID. Actual error strings are neither printed nor interpreted beyond those predicates. A zero-error sweep report does not mean the extraction queue has zero parked failures.

Every removed pending record in each adjacent interval has a committed episode with equal ID, source, source_ref, occurred_at, cwd and cwd_is_repo. Every changed pending input retains all non-retry fields; hashes/lengths account for each added input. Queue-input preservation covers the captured distilled inputs, not upstream transcript reconstruction.

The deployment note identified by ID hash `3500d0f460a1daa4d42ab978f2ecaaa479c9fa6af3ac73a0ffbb7893c4a3ac80` is present in the 02:14:53 queue with 750 text bytes, enqueued 02:14:53.182399 UTC. Its complete payload SHA-256 `734399a9c8bf1786bb360cb8054d4c8f26e457c835e34b778170be6b43602079` is byte-identical at 02:17:07. It remains unparked, attempts 0, and has no committed episode. This independently proves frozen queue persistence, not final extracted-fact durability or sub-second p95. No retry was attempted.

## Structural failures and ownership checks

Names and aliases are normalized and deduplicated per entity. Pure slug fallback is excluded from listed-spelling ownership defects. The actual 02:03:46 baseline is freshly computed, including 2,970 entities with no current facts; it is not the older 2,969 baseline.

| Structural measure | 02:03:46 | Each later snapshot |
|---|---:|---:|
| dangling fact-endpoint occurrences | 2,441 | 2,441 |
| entities with no current or historical fact | 2,851 | 2,852 |
| entities with no current fact | 2,970 | 2,972 |
| listed spellings with missing claim | 0 | 0 |
| wrong-owner listed-spelling occurrences | 504 | 504 |
| unlisted owner claims | 29 | 29 |
| spellings listed by multiple entities | 462 | 462 |
| spellings listed across multiple types | 27 | 27 |
| dangling claim owners | 0 | 0 |
| self-loops, all/current | 1,099 / 93 | 1,099 / 93 |
| missing episode refs / missing or extra reverse adjacency | 0 / 0 / 0 | 0 / 0 / 0 |

Other than the two no-current additions and the one no-facts addition, complete hashed defect sets are exactly equal, not merely equal in count. All 41 new entities resolve through their own slug, and 50 newly listed names/aliases on new or changed entities resolve to the listing owner through the actual store lookup API. No owner was selected or transferred by this reviewer. The current relation set remains exactly the documented 39 relations, with zero noncanonical current facts.

The new no-facts entity is a runbook, entity key SHA-256 `137a325eadf18b5a66c0eb373c4af821754eacf51de4e3a8cc8b13b95675bbee`, payload SHA-256 `b7a0d89c8e1fea82e6e47b464c0620dbb385d05248e8ff2a915c853afebe123a`. It has no aliases, one correct name claim, one repo ref and a 127-byte description. CreatedAt and LastSeen are 01:27:36.275 UTC. It is absent at 02:03:46, present with zero touching facts at 02:09:13, and remains a no-facts entity in both explicit sweep snapshots. The structural defect hash is `0391ee500b5c6f2f731a8be62e9121d26c7114470f9be2c3d8b35d5882a108c6`. This is an actual graph-quality regression, not concealed under the prior startup-refusal PASS.

Exactly one newly ingested episode shares its occurred_at timestamp and has cwd equal to its repo ref: episode ID hash `a0ebec7d2d5b2d1aefdbfeea4824236b61a9c445652337e43f830462e5506f6d`, ingested 02:08:30.102489 UTC. The entity has no direct episode field and no alias attestation, so this is a timestamp/cwd correlation, not a proven causal link. The source's `applyWith` resolves declared entities before facts; `resolveEntity` can persist a newly admitted entity without requiring a touching fact, and the final transaction does not check for hollow entities. That is a plausible existing normal-write mechanism, not an observed trace of the original extraction response. No semantic status/value classification was inferred from its name.

The other new no-current entity is a decision with one historical fact, not another zero-fact hollow. It has the same creation-time correlation and a direct alias-attestation episode link. Historical-only entities may be legitimate; no retirement or ownership decision is proposed for it.

## Retrieval measurements

All twenty actual installed-artifact benchmark CLI runs succeeded against the four frozen private stores. Each snapshot also ran the independently source-built individual-rank helper for all 235 questions, including misses. Every complete raw map remains byte-identical after its five-suite benchmark and rank runs. Installed and retained host hashes were checked again after the full matrix and still match the values above.

| Suite | Hits at all four snapshots | Mean answer rank at all four | Maximum payload bytes: 020346 / 020913 / 021453 / 021707 |
|---|---|---:|---|
| heldout-2026-09-03 | 51/62 | 4.8431372549019605 | 12,112 / 12,110 / 12,110 / 12,111 |
| heldout-b | 29/66 | 5.068965517241379 | 13,366 / 13,359 / 13,359 / 13,369 |
| probes | 7/7 | 1 | 9,962 / 9,974 / 9,974 / 9,961 |
| tuning-strict | 45/50 | 4.511111111111111 | 11,533 / 11,543 / 11,543 / 11,542 |
| tuning | 47/50 | 3.978723404255319 | 11,533 / 11,543 / 11,543 / 11,542 |

All 235 individual answer ranks are equal across all four snapshots. Full miss membership, ranks and hashed fact previews are also equal for every suite. All cap counts are zero at 24,576 bytes. Payload lengths do change: compared with 020346, 219 questions differ in length on both 020913 and 021453, and 221 differ on 021707. Accordingly, complete non-timing benchmark objects are not equal across the entire interval; only the measured retrieval success/rank/miss properties remain equal, while payload means/maxima stay below the cap. No benchmark file or expectation was changed. The original 53/62 and 34/66 held-out floors still fail; the other three original floors pass.

`retrieval-summary.jsonl` records all full comparison fields. A first run of its private formatting script rejected an omitted probes miss list (`undefined`); the script was corrected to normalize omitted empty lists to `[]` and then completed. No benchmark was rerun or altered for that reporting-only correction.

## Reproduction and evidence

Build all three helpers in this private export:

```sh
go build -o grade ./cmd/actual-grade
go build -o evidence ./cmd/independent-evidence
go build -o drift ./cmd/independent-drift
```

For each table row, run `./evidence restore <full-backup-path> s<TIME> <backup-sha256> > restore-<TIME>.json`; destination must not exist. `run-audits.sh` contains exact serialized per-store commands for all four delta/audit/closure/queue/supplement/drift comparisons and the four full benchmark/rank runs. The additional `./grade supplement s020346 s021707 > final-supplement.jsonl` checks every newly listed spelling and the two no-current additions; `./grade conflict-closure s020346 s021707 > final-attestation-closure.json` verifies changed attestation provenance. `node summarize.mjs > retrieval-summary.jsonl` compares all 235 individual ranks, full miss membership/ranks and all aggregate non-timing fields.

`ARTIFACTS.sha256` covers all private authored/copied helper source, command scripts, report and completed JSON/JSONL evidence. Hashes identify preserved evidence; no reports supplied by the lead were substituted for independently executed measurements.

## Limits

This review fails final graph quality and does not certify the full goal. It does not grade status/value semantics, the external-name guard corpus, fifty new held-out questions, live remember p95, Kimi/OpenCode orient coverage, final hygiene no-op, historic recovery, credential remediation, or two complete fresh-context rounds of every goal clause. The pending deployment note is not yet independently proven ingested. The repository-reference loss and surviving older structural defects remain disclosed.

Preservation concerns every active logical record and its bytes in these complete backups. It does not prove physical Badger-file equality, deleted/expired keys, prior internal LSM versions, resident-process memory hashes or crash/disk-fault recovery. Reads can rewrite physical private Badger files while preserving all active logical records. All inputs and private evidence remain available; this reviewer proposed no live cleanup and performed none.
