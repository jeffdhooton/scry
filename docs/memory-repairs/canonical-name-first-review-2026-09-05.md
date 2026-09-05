# Independent canonical-name code/replica gate: FAIL

Exact candidate `a9651772576f615586b2236b1f9ad47279d73293`, parent `dd76511`; old production behavior reproduced using an independent `393eeec` archive. Review date 2026-09-05. This is a bounded canonical-name resolution gate, not a full-goal or deployment certification. Do not deploy this candidate before the proved boundary failures are corrected and independently regraded.

## P1: canonical-name equality chooses an index owner across existing retained homonyms

At `internal/memory/resolve/resolve.go:233`, canonical normalized-name equality now bypasses incompatible-type refusal. The preceding exact-name veto only checks the natural storage slug. It does not detect another existing identity with the same canonical normalized name under a different retained slug.

Independent fixture:

1. Existing project `{Slug:"atlas-project-id", Name:"Atlas", Type:"project", Description:"software project"}`.
2. Existing machine `{Slug:"atlas-machine-id", Name:"Atlas", Type:"machine", Description:"physical machine"}`.
3. Both have prior facts to an existing `telemetry-engine` tool. Neither has natural slug `atlas`. The `Atlas` index claim belongs to the project.
4. Apply one new episode explicitly declaring `{Name:"Atlas", Type:"machine", Description:"The existing physical machine"}` and fact `{Src:"Atlas", Relation:"runs_on", Dst:"telemetry-engine", Fact:"The Atlas physical machine runs on the telemetry engine.", Confidence:0.95}`.

Old `393eeec`: `ErrAliasClaimed`, with identical whole-graph state before/after. It does not choose an owner for the ambiguous name.

Candidate: successful Apply, `EntitiesUpdated=1`, `FactsAdded=1`; the machine assertion is stored as **`atlas-project-id runs_on telemetry-engine`**, attached to the project. This is new cross-type fact routing through the name index, despite two established distinct identities. The project remains typed project and the machine remains present, so this is misattribution rather than literal entity-record deletion; that does not make the routing safe.

Fixture construction uses `ClaimAlias` only to represent legacy ambiguous state: put project; claim `Atlas` for the prospective machine slug; put machine; claim `Atlas` back to the project. No Apply call receives authority to change alias ownership. This is a fixture, not a proposed live repair. Ordinary PutEntity's current conflict prevention does not make existing ambiguous stores or reviewed retained-slug states impossible.

Reproducer: `TestIndependentCanonicalTwoRetainedHomonyms` in `new/internal/memory/resolve/independent_canonical_test.go`. `new-tests.log` fails; `old-tests.log` safely passes this disproof check. An independent restored-live inventory found **zero current examples** of this exact shape (cross-type canonical-name homonyms without a natural exact entity). The finding is a prevention regression proved by independent existing-state input, not a claim that this specific corruption already exists live.

The natural-slug precedence case still passes when one existing entity occupies `atlas`; the failure specifically requires both identities to have retained slugs. A complete uniqueness check for the exceptional canonical-name path must refuse ambiguity rather than infer ownership from the index or type.

## P2: a generic canonical role phrase now accepts mismatched identity metadata

Independent fixture:

- Existing entity `{Slug:"cedar-retained", Name:"the machine", Type:"project", Description:""}`.
- One extracted declaration `{Name:"the machine", Type:"machine", Description:"Should not replace existing identity"}` at `2026-09-05T22:00:00Z`.

Old `393eeec` refuses the conflicting mention and leaves metadata untouched. Candidate succeeds, retains project type, fills the project description from the machine mention, and updates LastSeen. The extracted generic role has thereby been accepted as the project identity.

`roleNames` in `aliases.go` explicitly rejects `the machine` as alias vocabulary. The earlier generic checks in `resolveEntity` do not cover this phrase, and the old incompatible-type refusal previously stopped this fixture. Bypassing that refusal based only on canonical-name equality exposes it. This is not evidence that the patch changed the generic word tables; it demonstrates that its new exception makes an uncovered generic case newly reachable.

Reproducer: `TestIndependentCanonicalAliasAndCollisionGuards/generic-canonical`; `old-generic.log` PASS, `new-tests.log` FAIL. The failure uses a legacy bad canonical record and does not assert that a legitimate machine is named by this phrase.

## Actual post-merge replica: intended SQL-file fix does work

Independently verified and repeatedly restored source:

`/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger`

SHA-256 before/after: `5323fef7b594c748fd61462909cc808f78c1bd0a104da070abdc54cb21e102aa`.

Each independent restore has 80,203 facts, 30,436 entities, and 51,749 alias-index claims. The actual retained owner is:

- Slug `0160-task-evidence-rulessql`.
- Canonical name `db/migrations/0160_task_evidence_rules.sql`.
- Type `tool` and the reviewed migration-file description, six aliases, and two repo references.

On old `393eeec`, synthetic exact-canonical declarations typed `project`, `machine`, and `service` fail with the alias claim pointing to the survivor while the resolver attempts `dbmigrations0160-task-evidence-rulessql`. `tool`, `concept`, and protected artifact `value` inputs succeed.

On the candidate, all six declarations succeed. Each probe applies two explicit synthetic current facts, exercising the canonical name as both source and destination. Both route to the retained survivor. All 80,203 original current and invalidated facts remain identical; all 51,749 alias claims remain identical; every entity's metadata remains identical apart from the survivor's expected LastSeen refresh to the synthetic episode time. The survivor's reviewed type, name, description, aliases, creation time, and repo references stay unchanged. No duplicate natural-slug entity appears.

These probes reproduce a matching failure mechanism and its correction. The provider's actual intermediate extraction was not retained, so **the real parked episode's declared type is unknown**. This report does not claim to replay its exact provider output.

## Other passing boundary checks

- Existing exact natural-slug identity wins over a retained canonical index owner without changing global alias ownership.
- Noncanonical cross-type aliases refuse atomically.
- Typed mentions through concept-only aliases refuse atomically.
- Stale index claims refuse without mutating the owner.
- Punctuation/natural-slug collisions whose canonical normalized names differ refuse without rewriting the entity.
- A unique canonical concept under a retained slug upgrades to the typed identity, preserving its existing description; space, underscore, case, and hyphen-normalized spellings behave consistently without new entities.
- One episode declaring two distinct existing machine/project identities with each other's names as proposed aliases does not steal either alias claim or fuse their fact endpoints. Both entities retain their reviewed metadata aside from normal LastSeen refresh (`one-episode.log`).
- Full committed suite: `go test ./... -skip '^TestIndependentCanonical'`, exit 0 (`full-suite.log`). Passing committed tests do not cover the independent retained-homonym or generic-role failures.

## Evidence and scope

Root `/tmp/scry-canonical-independent-sep05.Scu1K1` contains exact `new/` and `old/` git archives, independent tests, restore directories, and reports.

- `new-tests.log` SHA-256: `a4b2e71aabba6132dfaa23f57680c1bee1d0f0a284c91094cddc07c4e831ae33`.
- `old-tests.log` SHA-256: `f0692aa112416c0293b8eab0c794480150d89ca3596ec437a6eb50832b5ffbbe`.
- `old-generic.log`, `one-episode.log`, `full-suite.log`.
- `live-canonical-homonyms.json`: empty independently measured inventory, no historical rewrites.

The goal house rules were read in full earlier in this continuous review session. All synthetic writes, tests, archives, and reports stayed in review-owned temporary paths. No live-store writes, deployments, model calls, remember calls, historical repairs, or shared-repository edits occurred. The source backup was read-only and remained hash-identical.

The gate remains FAIL on the two concrete newly accepted inputs above. The success of the narrow actual SQL-file probe does not authorize the broader exception.
