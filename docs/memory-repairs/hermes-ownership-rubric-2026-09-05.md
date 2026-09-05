# Hermes ownership review rubric

This is a review contract, not an apply manifest or a completed ownership audit.
The three identities remain distinct. No owner is inferred automatically from
a spelling, relation, repository cwd, entity type or sentence keyword.

## Scope and evidence

The restored Mini source `memory-20260905T195752Z.badger`, SHA-256
`7261cfff0a7957d825855c430dbbc0f333517e806b7c47d8fc03a66fe8fc1548`,
contains 856 unique current/historical facts touching these identities:
331 touch `hermes-ops`, 306 `hermes`, and 237 `mac-mini`. Overlapping endpoint
counts must not be summed. The subsequent six-record migration repair did
not affect this inventory. All facts, entities, episodes and alias claims
are exported in `/tmp/scry-migration0160-fresh-sep05.teYtyC/review-inventory/`;
the reproducible read-only helper is in its sibling `code/cmd/review-inventory/`.
Any new live manifest needs a fresh complete inventory and independent review.

For each unique fact record, read its full payload and all available source
episodes, then any relevant companion facts and independently attested local
artifact. Record a per-fact decision with exact old key/hash, proposed source
and destination (or unchanged literal), evidence, uncertainty and disposition.
An unchanged decision also needs a reason. An incomplete review is not a keep.

## Identity boundaries

| Identity | Belongs here | Insufficient evidence |
| --- | --- | --- |
| `hermes-ops` | Repository/code, checked-in scripts, specs and deployment artifacts as repository contents | The current contaminated agent description; a session cwd; generic “ops” or “Hermes” |
| `hermes` | The standing gateway/agent's behavior, runtime settings, integrations and scheduling | A host address or deployment path without an agent subject |
| `mac-mini` | Hardware, operating system, host resources, machine-scoped processes and installed services | An agent using a service, or a model's behavior, merely because it runs on this host |

Facts may belong outside all three. The review must not force a wrong edge
into the nearest of these identities. A source can be correct while its
destination is wrong, or both may be wrong; evaluate endpoints independently.
An ambiguous “Node host” is not automatically the Mini. Local repository
existence proves the repository, not historical runtime deployment.

## Preservation and abstention

- Preserve every current and invalidated fact's text, relation, raw relation,
  literal, validity, confidence and full episode list. Never nudge timestamps,
  overwrite an occupied key, discard a duplicate, or revive invalidated facts.
- Proposal/recommendation facts remain proposals. Relocation must not promote
  intended configuration or a replica result into an actual deployment.
- Discussion of a mistaken memory record is not automatically a fact about
  the runtime. Keep audit subjects and the thing audited distinguishable.
- Conflicting episode evidence, multiple plausible owners, or unsupported
  relation semantics receive `UNRESOLVED`, with concrete alternatives and
  missing evidence. This does not authorize a guessed move or silent keep.
- Metadata and alias cleanup are separate explicit reviews. Do not use fact
  relocation as entity merging or leave an exact-name lookup hollow.
- Exact-key conflicts are explicit unresolved repairs. Preserve both records;
  neither identical words nor a newer timestamp licenses deletion.

## Initial evidence examples — not applied

1. `agent-template located_at hermes-ops`: keep the repository endpoint.
   Episode `1041b44d0bfa6fc15b746a8fe4716162fcda4fbbe782bcf98bce01ab2acd033a`
   explicitly proposes a folder/unit in the repository. The existing sentence
   sounds implemented while the episode says recommended; record that separate
   factual-status caveat rather than turning this into a runtime/host edge.
2. `advocates deployed_on hermes-ops` at `2026-06-12T19:10:42Z`: the sentence
   explicitly names Forge and `/home/forge/www.theadvocates.org/current`.
   Its source episode is in the Advocates repository. A possible destination
   repair must review the exact Forge identity and key conflict, not choose
   Hermes or Mini. No move is approved by this example.
3. `childscribe-laravel deployed_on hermes-ops` at `2026-08-19T15:45:00Z`:
   both sentence and episode mention Laravel production on Forge. The clause
   “tracked in hermes-ops context” does not make the repository a host. Review
   the exact Forge destination separately before proposing a move.
4. The same triple at `2026-09-01T06:40:17.72Z` says “Docket is deployed on
   the Node host.” The episode is a Docket host-parity session. The current
   ChildScribe source is unsupported, but the Node host identity is not yet
   established. Mark unresolved; do not change only the source and present the
   remaining repository-as-host endpoint as corrected.

Completion requires all 856 snapshot records (plus later drift) to have their
own reviewed disposition, followed by exact-manifest replica and live checks.
These four examples do not satisfy that bar.
