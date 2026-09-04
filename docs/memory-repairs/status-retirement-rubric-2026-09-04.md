# Status-node retirement rubric — 2026-09-04

This is the semantic review rubric for the candidate inventory in
`status-retirement-candidates-2026-09-04.json`. The inventory is not an apply
manifest. Exact entity, fact, alias, and adjacency fingerprints will be
regenerated from a fresh Mini backup only after the extraction queue is stable.

## Boundary

A retirement candidate is a verdict, state, outcome, or transient repository
condition that answers a question about another identity. It is not retired
merely because it has status-like words. Named policies, tests, runbooks,
protocol markers, events, and other independently referable things remain
entities.

Every current and invalidated touching fact is preserved. Fact text, relation,
raw relation, validity timestamps, confidence, and episode provenance remain
byte-for-byte equivalent after JSON round-trip. No rule infers a new owner from
the spelling or sentence during apply; the choices below are the reviewed
manifest inputs.

## Replacement rules

1. An edge whose destination is the retired status node becomes an attribute on
   the same source. Its literal value is the retired entity's exact display
   name, preserving case and punctuation from the reviewed snapshot.
2. An edge or attribute whose source is the retired status node is relocated
   only to an explicit owner reviewed for that individual fact. Different
   source facts on one generic status may have different owners. An existing
   attribute keeps its literal unchanged. An edge keeps its non-retired
   destination unchanged.
3. Hollow candidates need no replacement facts, but their complete entity,
   alias-index, and adjacency snapshots are still fingerprinted.
4. Any stale, malformed, or nonempty reverse-index record requires its own
   exact key/hash/reason acknowledgment in the apply manifest.
5. A missing owner, key collision, self-loop, external alias listing, snapshot
   drift, or incomplete fact/adjacency review aborts the whole manifest.

## Explicit source relocations

| Retired source | Reviewed owner | Reason |
|---|---|---|
| `validation-failed` | `scry` | The sentence describes Scry's `Apply`/alias-resolution defect and keeps `contextual-enum-identity` as the unchanged destination. |
| `created` | `docsdemoreadmemd` | The fact is the creation record for `docs/demo/README.md`; the existing literal `docs/demo/README.md` remains unchanged. |
| `published` | `seed-published` | The `+13.0%` attribute is explicitly the result of the published benchmark seed. |
| `all-gates-passing` | `statelicenselookup` | Companion facts from the same episode identify State License Lookup at commit `0012ca7`; the existing commit literal remains unchanged. |
| `gate-green` | `docket` | The task-gate sentence reports Docket's DO/DB suites; the existing task literal remains unchanged. |
| `tests-passing` | `docket` | The operations API and database gate is a Docket repository result; the existing `loop/operations-domain` literal remains unchanged. |

The first semantic pass also found four source-bearing candidates whose owners
are not yet reviewed: `changes` (four outgoing facts across unrelated
contexts), `gates-passing`, `updated`, and `clean-slate`. Their manifests are
intentionally incomplete until every outgoing fact receives its own evidence-
backed owner. A single owner chosen for the status spelling would be an
unreviewed store-scale inference.

## Explicit exclusion

The following status-shaped nodes remain identities under the boundary above:

| Entity | Protected identity |
|---|---|
| `no-dashboard-automation` | Lasting operating decision against browser-driving Cloudflare and Google dashboards. |
| `north-star-wave-lands-clean` | Named north-star run/event that produced `wave-merge-playbook.md`. |
| `unmatched-photo-retained` | Durable behavior contract tested by `slot-photos.test.ts`. |
| `reconciliation-not-settled` | Named live-path equality assertion/invariant implemented by target-based reconciliation. |
| `tolerates-missing-optional-fields` | Durable generator behavior locked by `journal-generator.test.ts`; relocating it would reverse the `locks_contract` meaning. |
| `recap-emails-not-shipped` | Lasting ChildScribe product policy. |
| `all-attempts-success` | Durable posted-transition invariant. |
| `outputs-field-required` | Durable function/API signature contract. |
| `restart-survival` | Named “Gate 5 test” in its entity description. |

Malformed relations on these identities, if any, require a separate reviewed
fact repair. Status retirement is not a license to erase a test, contract,
policy, run, or invariant.

## Semantic review result

The first complete replica review disproved the 65-item preliminary inventory:
eight entries were protected identities and 69 current status/outcome nodes had
been omitted. The corrected inventory therefore contains 126 candidates: 57
retirements retained from the preliminary list plus 69 additions. It is still
not an apply manifest. In particular, source-bearing replacements remain
blocked until their per-fact owners are reviewed.

## Execution sequence

1. Deploy the independently verified prevention and retirement code to
   byte-identical laptop/Mini binaries, retaining both previous binaries and a
   verified nonempty Mini store backup.
2. Let the extraction queue reach a stable state, then take and restore a fresh
   Mini backup into a disposable replica.
3. Regenerate the candidate inventory and complete apply manifests from that
   snapshot. Recheck this rubric against every fact and record any additions or
   exclusions explicitly.
4. Dry-run the complete manifests on the replica, copy only the returned exact
   fingerprints, apply there, and prove relation count, fact count/content,
   endpoint, alias, collision, and second-pass convergence postconditions.
5. Give the replica result to a fresh-context disproof reviewer. Only after a
   clean verdict, regenerate fingerprints from the live store, dry-run, back up,
   and apply small reviewed batches.
6. Run both real sweeps and the complete benchmark/audit sequence. Any recreated
   candidate or proposed second-pass repair reopens the implementation loop.
