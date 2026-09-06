# Independent first real postguard sweep review

Verdict: bounded PASS for raw preservation, retained input, structural non-regression under the actual Name+Aliases contract, successful postdeployment extraction, and the five unchanged offline recall controls through the 2026-09-06 01:03:09 UTC backup. No semantic status-value PASS, whole-source cleanliness, hygiene convergence, or final-goal PASS is granted.

I ran the required orientation, read the active goal, the actual deployment review and the fresh-source BLOCK, and independently archived production commit `24eafab441f743da9e27ab1a72debe18c59975c5`. I inspected the preceding independent restore/delta helper and adapted a private copy. Every new helper, restore and report is under `/tmp/scry-first-sweep-independent.flj8Ff`. I performed no shared edits, live writes, provider calls, queue retry, transcript access, credential actions, configuration changes, deploy, room post or durable note. Graph identifiers, assertions, values, source paths and previews are represented only by digests. This first-sweep report is separate from the later second-sweep extension.

## Inputs and complete read preservation

Both source files were independently read completely and SHA-256 verified before direct Badger Load into new private directories. All raw key/value families were captured before candidate Store.Open. AllFacts, Entities, complete offline index construction, an actual recall read and Close preserved each entire raw map exactly. Missing keys are distinguished from present empty values. Sorted raw digests use big-endian uint64 key length, key, uint64 value length, value.

| Snapshot | Backup bytes | Backup SHA-256 | Raw records | Raw-map SHA-256 |
|---|---:|---|---:|---|
| Base 00:54:30 | 74,324,551 | `4aa1e86c349dd9f2d972a1b9567de7614bcc5736e3b3cae1e9a1f7cb8153fb9e` | 244,499 | `622c1366946552f97c83b74d896ef606498c11b5f9997b8356c8c21430354c2f` |
| First 01:03:09 | 74,430,346 | `70247d3efb58f6fc2d69768ff24d4b59d304c67c06ce6d2bd4fa00cabef3b208` | 244,694 | `07aacc2b661cfe8d7973b5d2bed50d4ab77e1f296f5512f25139bf660779a4e6` |

Both backups are named `memory-20260906T<time>Z.badger` under `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/`. The candidate installed laptop CLI independently hashes to `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`. The earlier independent actual-deploy report establishes both installed machines and 00:49:06 UTC process starts; this review does not repeat that remote process check.

## A real subsequent sweep and completed extraction

The persisted last-sweep timestamp advances from 00:34:14.783502 to 01:01:21.255638 UTC, after the verified deployment. Its complete stored report records 2,378 scanned files, six ingested files, six episodes, zero errors, and source counts Claude 2 / Codex 4. Startup is not counted as a sweep.

Four new stored episodes appear: one manual episode ingested at 00:57:56.223020 UTC and three transcript episodes ingested at 01:01:26.429120, 01:02:40.458638 and 01:02:41.923373 UTC. Last successful extraction advances to 01:02:42.094668 UTC. The earlier 01:01:26.634216 worker-completion timestamp was supplied by the root; the snapshot independently preserves the corresponding completed episode and the later extraction timestamp. New completed episode sources are one Claude, two Codex and one manual. No reenactment or provider probe was needed.

## Every raw-family delta and old assertion preservation

| Family | Added | Changed | Removed |
|---|---:|---:|---:|
| adj | 36 | 0 | 0 |
| al | 27 | 0 | 0 |
| ar | 0 | 0 | 0 |
| att | 25 | 0 | 0 |
| cur | 6 | 0 | 0 |
| en | 25 | 7 | 0 |
| ep | 4 | 0 | 0 |
| fa | 49 | 1 | 0 |
| meta | 0 | 4 | 0 |
| pq | 4 | 0 | 0 |
| rs | 0 | 0 | 0 |
| rt | 0 | 0 | 0 |
| ve | 19 | 3 | 0 |

All 81,057 old fact assertions survive: 81,056 raw records are exact and one gains only InvalidAt, at a time not before ValidFrom. Its key SHA is `69ba95b23d80ce74382e4d910b20d713d59a9c14bc7b84aa4f8099f28be2cd78`, prior payload SHA `0f6d764ee6e4b0d38286c2a672d28f9853746de188fff77f07d92cfef9b47806`, later SHA `b02ca74161f6fdc90c9a96497ef059a0fd9e27296a0aee7df3637e784ca577fb`. Its text, relation/raw relation, source/destination/value, validity start, confidence and provenance remain exact. The invalidation is structurally preserved, not semantically endorsed. All 7,845 preexisting historical facts are raw-byte identical.

Every one of the 49 new facts has nonempty provenance, cites a newly stored episode, has complete episode closure and has existing entity endpoints. The fresh graph contains 81,106 facts: 73,259 current and 7,847 historical. It still uses exactly all 39 documented current relations, with zero noncanonical current facts.

All 42 rejection/retirement markers and all thirteen previously parked payloads survive byte-for-byte. No old episode or alias claim changes or disappears. Four new pending records contain one Claude, two Codex and one manual input, all attempts zero and nonparked. The root's note identified in the request is present exactly once in pending and absent from completed episodes; enqueue time is 01:00:11.836015 UTC. Its raw pending payload SHA is `812c84a9c2be4c0a724bf6c441a33acde1b109a3ea544b06c68ba607c180715b`. This independently proves durable pending state; the reported 97 ms acknowledgment is a supplied single observation, not an independently measured p95 or completed extraction.

The seven old entity changes comprise two last_seen-only, four last_seen/repo_refs, and one last_seen/aliases update. All old aliases, names, types and descriptions are preserved. Three repository-reference changes are additions; one six-entry list adds a reference and evicts one old reference. The exact eviction is reproduced by unchanged `resolve.AddRepoRef`, with entity-key SHA `d3e451090715b3fca99941618b86d190c17c0d3a80bdb7d8b714c53265b22e64`. Thus a claim that all old repository metadata is preserved would be false. This is the existing six-reference cap, not an entity merge or reviewed identity transfer. Complete reference hashes are retained in `sweep-details.json`. Three old value-evidence records append episodes/spellings only, retaining every old episode and spelling and complete episode closure.

## Complete structural sets and the slug-only correction

The proper listing audit uses exactly entity Name plus Aliases, normalized by production Store.Normalize. Its complete before/after sets are identical: zero missing claims; 504 wrong-owner listing occurrences; 29 index claims not listed by their owner; 462 multiple-owner listings; 27 multi-type listings; zero dangling index owners. These are existing defects, not a clean graph.

The older broad inventory additionally treats the storage Slug as an alias spelling. Its apparent missing-claim increase, 3,876 to 3,881, consists entirely of five new slug-only entries. All 3,881 rows have is_slug=true and is_name=is_alias=false. PutEntity deliberately indexes Name+Aliases; daemon `resolveMemorySlug` falls back to Slugify when the alias index misses. Consequently these rows do not prove missing listed aliases or a broken slug lookup. Every one of the 25 new entities succeeds in direct slug lookup, and all 26 actual names/aliases resolve to their owner. These reads preserve the whole raw map. The broad wrong-owner and unlisted counts are 505 and 28; the proper contract counts above should be used for listed-alias claims.

All remaining complete structural sets are unchanged: 2,441 dangling endpoint occurrences; 2,851 entities touching no facts; 2,969 touching no current facts; 1,099 self-loop keys, including 93 current; zero missing provenance episodes, missing adjacency or extra adjacency. Complete set digests and every added/removed hash are retained. No new hollow, dangling endpoint, listed-name/alias error, multi-type collision or self-loop appears in this interval. This does not classify new identities semantically or prove production hygiene has no proposals.

## Independent unchanged-suite controls

I executed the installed candidate's actual `memory bench` offline CLI against each frozen replica with all five unchanged archived files and top=20. All ten commands passed and preserved both complete raw maps. Hit counts, mean answer ranks and complete ordered missed-question/rank lists agree across the two snapshots. Graph previews are hashed.

| Suite | Base → first | Mean rank, both | Maximum bytes, base → first |
|---|---:|---:|---:|
| heldout-2026-09-03 | 51/62 → 51/62 | 4.8431372549019605 | 12,108 → 12,107 |
| heldout-b | 29/66 → 29/66 | 5.068965517241379 | 13,373 → 13,373 |
| probes | 7/7 → 7/7 | 1 | 9,984 → 9,960 |
| tuning-strict | 45/50 → 45/50 | 4.511111111111111 | 11,556 → 11,541 |
| tuning | 47/50 → 47/50 | 3.978723404255319 | 11,556 → 11,541 |

All cap-violation counts are zero under 24,576 bytes. The original first two suite floors remain unmet. Equal aggregates do not prove identical successful-query ranks or full payloads. No new fifty-question set was written.

## Boundaries and reproduction

This is a measured first-sweep structural/preservation and recall-control PASS only. No lexical naming heuristic is used to claim the new entities are semantically valid; context-based status/value classification remains unverified. The known earlier overwritten history and credential persistence remain unresolved. Existing structural defects, proposed hygiene repairs, recall floors, final semantic identity review, remember p95, coverage/orient and two complete fresh grading rounds are outside this PASS.

Private helper source is `cmd/actual-grade/`. `restore SOURCE NEW_DIR SHA256` independently loads and checks complete preservation; `delta before after`, `sweep-audit before after`, `sweep-details before after`, `sweep-closure before after`, and `proper-claims before after` produce the first-sweep evidence. `sweep-bench before after ROOT /Users/jeff/go/bin/scry` produced `bench-safe.jsonl`. The complete delta contains every added or changed key/payload digest across all raw families. `ARTIFACTS.sha256` pins reports, helper source/binaries and safe evidence.
