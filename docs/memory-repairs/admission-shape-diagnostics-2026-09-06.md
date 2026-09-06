# Prospective assertion-shape defects: private diagnostics

These are reproductions of existing behavior, not a proposed live repair or
passing acceptance test. Deployed-source d1f0a958's resolver was exercised with
provider-free synthetic episodes in disposable stores. No shared resolver or
queue code was changed; no stored source fact was corrected.

The synthetic identifiers Velatrix, Lyrion and Zyrion do not occur among any
name, slug or alias in the complete 23:51:34 source entity inventory.

## Reproduced defects

- A `status` assertion with both mentions resolving to the existing Velatrix
  identity becomes a current `related_to` self-loop with raw relation status.
  The fact sentence describes pending deployment review, not a relationship
  between two identities.
- A canonical `calls` assertion about real recursion also becomes a current
  self-loop. Here the sentence is meaningful: the service recursively calls
  itself until its input tree is empty. Structural reflexivity alone does
  not establish that the assertion is useless or false.
- Different reviewed spellings of the same identity can yield a self-loop
  even when the extracted source and destination strings differ.
- A `has_worker_count` attribute with value 12 maps to canonical status.
  A later status=in-review assertion invalidates that capacity observation,
  although its sentence explicitly says no capacity change occurred. The
  old row remains stored historically, but its asserted validity is wrong.

The current resolver first maps relations to the closed vocabulary. For
status pointing at a known identity it preserves an edge by mapping to
related_to; after endpoint resolution it does not reject self-loops.
Exclusivity compares source and canonical relation, not the actual attribute
dimension. No new model classification, ownership rule or lexical word list
is needed to reproduce either mechanism.

The frozen actual post-stops snapshot has 1,099 self-loop rows, 93 current.
Its three newest examples come from episode 2e4dd9f7: blog-no-cms decided
itself, blog-query related_to itself (raw status), and smoothhauling-blog
related_to itself (raw status). This is a structural observation over the
complete stored rows; it does not certify the model's original extraction
or authorize rewriting those rows. Their source was separately reviewed
under the stops-table ingestion-drift gate.

## Unsafe shortcuts and the unresolved representation contract

Do not apply hygiene's inferred self-loop invalidations. The existing code's
comment that every self-loop says nothing is disproved by the recursion
fixture. Do not invent another owner from the fact sentence. Do not widen
status exclusivity: that would increase unrelated invalidations.

Rejecting the entire incoming episode transaction could prevent a loop from
entering the graph, but is not a complete memory-quality solution. The queue
currently retains the original pending text, not the exact extracted Result;
its permanent-failure classification would also need explicit handling.
Parking every such episode would withhold its other valid assertions and
could fail ingestion/recall coverage. No such change is implemented.

A durable solution needs to distinguish an assertion about one subject from
an entity-to-entity edge, and distinguish independent state dimensions, while
retaining exact text, timestamps, confidence and provenance. The existing
Fact invariant is exactly one of entity Dst and literal Value; extending it
requires a separate representation, key-compatibility and temporal-behavior
contract. No new schema, asserted recipient, validity rewrite or migration
is approved by this diagnostic.

## Reproduce

Private test source:
`/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/code/internal/memory/resolve/admission_shape_diagnostic_test.go`.

Run `go test ./internal/memory/resolve -run TestAdmissionShapeDiagnostic -v -count=1`
in that private source archive. The tests deliberately assert the reproduced
current behavior; their PASS means reproduction, not correction. The full
command output is archived alongside this report. No provider, queue replay,
live repair, source-retention change or shared test locking in bad behavior.
