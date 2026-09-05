Independent frozen-snapshot graph audit, September 5, 2026

Verdict: graph-quality clause fails on this snapshot. This is an audit and bounded proposal packet, not an apply manifest or a final goal pass. There were no live writes, repairs, model calls, deployment operations, or memory remembers.

Snapshot: `/tmp/scry-qwen-live-sep05.Koj9v3/memory-20260905T182812Z.badger`, SHA-256 `24621f12ab33ad0d7b32f96238f1b7854430abf41da9c3ff8b362dd8c33ae851`. Code independently archived at `1af0d3bdb7019d4045e3512dde702f4fe74276f1` into this temporary directory. The source working tree was never edited. Both restores used newly created, nonexistent replica paths inside this directory; no other store was opened.

The independent fold/pair enumeration matches `resolve.CrossTypeCollisionCount` exactly: 489 cross-type pairs in 427 folded-spelling groups. Each pair records every contributing spelling, distinguishing name, slug and alias. These are spelling/entity-pair occurrences, not 489 unique pairs of entity slugs: a pair can collide at more than one folded spelling. `results/cross-type-collision-pairs-complete.json` and `results/collision-groups-complete.json` are complete, deterministically ordered inventories, not samples.

The snapshot contains 30,175 entities, 79,560 current plus historical facts, 9,282 episodes, and 51,434 raw alias claims. Results:

| Measure | Count | Complete inventory |
| --- | ---: | --- |
| No current or historical fact references | 2,854 | `hollow-all-history.json` |
| No current fact references (includes historical-only entities) | 2,972 | `zero-current-facts.json` |
| Current missing fact endpoint occurrences | 0 | `missing-fact-endpoints.json` |
| Historical missing fact endpoint occurrences | 2,441 | `missing-fact-endpoints.json` |
| Distinct historical facts with missing endpoints | 1,995 | `missing-endpoint-summary.json` |
| Distinct missing entity slugs | 994 | `missing-endpoint-summary.json` |
| Listed aliases indexed to another existing entity | 464 | `listed-spelling-index-anomalies.json` |
| Listed names indexed to another existing entity | 64 | same |
| Slugs indexed to another existing entity | 45 | same |
| Slugs missing from normalized alias index | 3,848 | same |
| Listed aliases or names missing from index | 0 | same |
| Raw alias claims whose owner entity is absent | 0 | `raw-alias-missing-owners.json` |
| Raw claims not listed as owner name, slug or alias | 28 | `raw-alias-not-listed-by-owner.json` |
| Current self-loop facts (additional observation) | 89 | `current-self-loops.json` |

Every missing endpoint occurrence is historical/invalidated: 1,978 source and 463 destination occurrences. Do not report these as missing endpoints in current facts. No fact was dropped, invalidated or relocated by this audit. Empty destination is an attribute value, not a missing endpoint. Self-loops and historical-only entities are retained in their original form.

Index interpretation: the 464 alias and 64 name foreign-owner observations are not semantically adjudicated by mere index ownership. The complete inventory gives the current owner, its existence and the literal/normalized spelling; ownership does not prove entitlement. Every listed spelling is also exported in `listed-spellings-complete.json`, and all raw `al:` claims in `raw-alias-claims.json`. Slug-only index absence is separate from a broken listed-name/alias invariant: `PutEntity` registers `normalizedNameSet(e.Name, e.Aliases)` and does not promise a distinct alias-index record for every slug. A slug remains the entity's storage key. Accordingly the 3,848 slug-only absences are reported without declaring 3,848 exact-lookup blackouts. None of those normalized slugs is also listed as a name or alias on its entity (verified in `verification.json`). The 28 raw claims lacking listed metadata may be retained compatibility claims; this audit does not assume they are authorized or stale.

Definitions and exclusions: the hygiene implementation folds case, non-alphanumeric punctuation, spaces and plurals; per entity it deduplicates folded forms. It excludes entities without any current OR historical fact reference, and only counts pairs whose trimmed lowercase types differ, with empty types treated as compatible. It does not give concept a wildcard. The independent inventory found 75 additional cross-type spelling pairs excluded solely because an entity is unreferenced; these remain completely listed in `collision-pairs-excluded-unreferenced.json`. Admission's `TypesCompatible` would exclude 263 of the official 489 pairs, leaving 226. The 263 are separately listed in `hygiene-pairs-excluded-by-admission-compatibility.json`; they are not subtracted from this audit. No special code-file, reference, artifact or identifier namespace exemption was applied. Such a policy would change the metric and require its own explicit review; stored entity descriptions and provenance, not strings resembling paths, determine semantic distinctions in the bounded review.

Bounded first-ten review: `first-ten-reviewed-proposals.json` selects folded groups lexically, caps each selected group's complete touching fact set at 40, excludes completed Qwen, and records every skipped earlier group. ChildScribe-specific or mixed identity groups `6`, `action`, and `adversarialgrader` are deferred; `adjustment` is deferred because its 231 facts / 143 episodes exceed this pass's bound. The full inventory still includes them. The selected groups are:

| Folded group | Evidence-backed conclusion |
| --- | --- |
| `0160taskevidencerulesql` | Same migration identity, but a third existing owner `migration-0160` must join the review closure. |
| `64` | Program Health issue #64 and Docket §6.4 are distinct; punctuation folding creates the collision. |
| `67` | Program Health issue #67 and Docket §6.7 are distinct; preserve and separately review one suspicious ChildScribe outside endpoint. |
| `69` | Cell Saviors PR #69 and Docket §6.9 are distinct; Cell Saviors also holds a demonstrably unrelated ChildScribe PR #68 deployment fact. |
| `adapterts` | Exact contract path unresolved: generic ImportAdapter versus its pricing use may be one shared interface or distinct artifacts. Hold repair. |
| `agenttemplate` | Deployable unit and dated implementation campaign are related but distinguishable; keep three invalidated facts. |
| `aggregaterunner` | Two records identify the same execute.ts module; the third identifies a contained runner seam. A whole-group merge would erase that distinction. |
| `aicalibration` | A deferred decision and the Settings host page are distinct; the section alias does not establish identity or a rightful rehome target. |
| `apiappts` | Same Docket api/src/app.ts file, independently linked by exact commit cf2f56b and change behavior. |
| `apigate` | Narrow API gate and task-specific two-command verification contract are distinct despite plural folding. |

Each selected `evidence-<folded>.json` contains complete entities, all touching current AND historical facts with their complete fields, all linked stored episodes with summaries/source refs, outside endpoint metadata, index anomalies, and any foreign index owner's full entity/fact/episode evidence. The first ten contain 86 direct touching facts, including three historical facts, plus one additional foreign-owner fact for migration-0160. Packet files for the first 30 lexical candidate groups support reproducible selection; only the stated ten are adjudicated. Provenance evidence here is the stored episode record, not a new reading of original transcripts or current application source. All unresolved recipient/type/naming decisions are explicit, and no survivor is chosen by fact count or lexical order.

Reproduction (from this archived source directory):

```sh
go run ./cmd/independent-audit /tmp/scry-qwen-live-sep05.Koj9v3/memory-20260905T182812Z.badger /tmp/CHOOSE-A-NEW-ISOLATED-OUTPUT-DIRECTORY
node review-evidence.mjs /tmp/CHOOSE-A-NEW-ISOLATED-OUTPUT-DIRECTORY
node decisions.mjs /tmp/CHOOSE-A-NEW-ISOLATED-OUTPUT-DIRECTORY
```

The program refuses to restore if its output's `replica` path already exists. Exports use read methods after restoration; raw Badger verification is opened with `WithReadOnly(true)`. The raw key/value SHA-256, framed with each key/value's big-endian uint64 length, remained `c6cb7523f4ba07fb61fc0ba3c6c3d96a6deee8fd4bf77bdeeed5fcddcecb6868` across 239,214 keys. A second independent restore in `repeat-results` produced byte-identical base inventories, as recorded in `results/verification.json`. This verifies audit no-op behavior, not convergence of a repair dry run: no repair was attempted.

`results/artifact-hashes.json` contains complete JSON artifact SHA-256s and sizes. `results/verification.json` records source script hashes, repeat-output comparisons, snapshot hash, current/historical fact totals and slug-index interpretation checks. No declaration of live graph cleanup or goal completion is made.
