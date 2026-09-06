# Independent unattached-admission design disproof

2026-09-06. Baseline: private `git archive a06cd7b` export at
`/tmp/scry-admission-disproof.zKuQqE`. No production source or existing test was
edited. Verdict: **NO-GO for integrating the proposed deletion/finalization
behavior; GO for a private, explicitly separate transaction-journal and
incarnation-isolation prototype with the invariants below.** This is a bounded
design verdict, not a whole-goal PASS or live-store finding.

I ran session orientation and read the complete active goal, complete private
PROPOSAL.md, and complete schema-refusal-two-sweeps-independent-review report.
The last report proves a new metadata-bearing zero-fact runbook. Its temporal/cwd
correlation does not identify the model response or prove the original execution
trace. Root's later inventory and tests were not used as independent evidence.
All examples below are newly authored synthetic fixtures, run independently.

## Reproduction

From this directory:

```sh
go test ./internal/memory/store ./internal/memory/resolve -run 'TestDisproof|TestApply_EmptySlugSkipped|TestAtomicWriteCommitsOrRollsBackAsOneUnit|TestApplyReleasesTransactionalCompactCaches' -count=1 -v
```

Result: both packages PASS. Eight new characterization tests establish the
counterexamples and one compatibility property; the three existing named tests
also pass without assertion edits. Their PASS means the described deficiency
was reproduced, not that any proposed implementation passed. No provider,
queue retry, live write, daemon restart, config change or deployment occurred.

## Concrete failures

| Candidate or assumption | Runnable evidence | Consequence |
|---|---|---|
| Delete provisional entity using DeleteEntity | resolve/admission_disproof_test.go:19 | A same-slug orphan name claim seeded before the transaction is deleted. Reversing the entity creation did not reverse just this episode's writes. |
| Keep att: records while deferring en:/al: | same file:42 | A first runbook observation leaves one attestation; a later machine incarnation at the slug obtains an otherwise unadmitted alias from that old evidence. |
| Solve orphan attestations only at transaction end | same file:80 | An attestation already present before the episode admits an alias during resolveEntity; an actual fact routes through that alias. The entity now has a fact and escapes any hollow filter. |
| Serialize only final materialized Entity | same file:99 | Two declarations of the same identity retain only the first description and neither rejected alias in Entity. Both separate alias attestations exist. Exact per-occurrence metadata must accompany the final materialization. |
| Refuse entire mixed episode | same file:124 | An unrelated metadata-only runbook withholds 1/1 valid assertions. Transaction rollback works, but neither assertion nor exact extracted metadata is committed. |
| Overlay hides provisional declarations from identity lookup | same file:154 | An explicit tool target of status becomes a related_to edge when visible; omitting it changes the same extraction into an exclusive status attribute. |
| Existing event buffering is sufficient after create/delete | same file:180 | An observer receives entity put even though GetEntity already reports absence in the committed store, then delete. A subscriber can temporarily publish the phantom entity. |
| Additive opaque bytes require numeric schema bump | store/admission_evidence_disproof_test.go:11 | Counterevidence: opaque bytes, including an unknown nested field and precise timestamp spelling, survive full backup, direct load, application Open, and an ordinary a06cd7b writer at the same schema. This does not prove older admission behavior. |

Paths in the table are relative to internal/memory. None of these examples
reinterprets a preexisting live hollow or infers an owner from its name. The
incarnation fixture deliberately contrasts two synthetic types to show that
slug-only attestation storage cannot prove they are the same identity; it does
not claim the current code merges two preexisting cross-type identities.

## Why the source produces these results

`resolve/resolve.go:87` creates the transaction facade; `:119` resolves every
declaration before `:134` resolves facts. `resolveEntity` at `:264` constructs
the new record, calls alias admission before PutEntity at `:285`, and writes
without asking whether any assertion will reference it. `:313` preserves only
the first nonempty description on repeated declarations. `ensureEntitySlug`
at `:833` creates endpoint stubs at `:857`; `resolveFacts` at `:540` deliberately
resolves both endpoints before the empty endpoint veto at `:558`.

`resolve/aliases.go:235` AdmitAlias writes/consults attestations before entity
persistence. For otherwise unowned unrelated aliases, its final threshold
branch near `:400` admits on distinct episode count. `store/pending.go:244`
keys this evidence only by slug and normalized spelling, stores up to eight
episode IDs, and records neither identity type nor incarnation. Therefore an
orphan's episode count can affect a brand-new identity immediately. Merely
restoring the old att: bytes at finalization preserves this vulnerability for
the next creation. Deleting them would destroy old provenance. Both operations
are insufficient without an earlier admission boundary.

`store/store.go:402` PutEntity already checks claim ownership, but a claim
pointing to the same proposed slug is permitted. `DeleteEntity` at `:1175`
deletes listed names/aliases and the normalized slug while they still point at
the slug. It has no transaction-start before-images, so it cannot distinguish
old orphan claims from newly introduced claims. Contrary to its broad comment,
it does not scan every al: key for other unlisted orphan claims. It leaves
attestations untouched. It is a maintenance primitive, not a provisional-write
undo primitive.

`store/store.go:236` AtomicWrite uses one Badger transaction and buffers events,
publishing them only after a successful commit. Nested AtomicWrite calls reuse
the facade. This is a useful existing overlay; copying the resolver into a
separate in-memory graph would require duplicating all Store lookup semantics.
The proposal can retain the existing snapshot-plus-staged-writes behavior.

Event buffering at `store/store.go:1426` does not coalesce opposite events.
`daemon/memory_queue.go:55` feeds put/delete individually into the search index;
`search/index.go:223` and `:231` independently lock each update. A normal
subscriber ultimately removes the fixture's phantom, so this test does not
prove a permanently stale daemon index. It does prove transient publication
and an observable put that never represented committed data. Suppression must
be inside the successful transaction's event plan, before any callback fires.

`resolve/resolve.go:91` always releases the facade's compact cache.
`resolve/aliases.go:925` caches by Store pointer; `:941` has a one-minute stale
test. Preserve that release on success, error and panic. Do not run new
resolution decisions after pruning provisional records: its compact name
snapshot can then contain names deliberately removed from the staged store.
The existing success/rollback cache-lifetime regression test passes here.

`resolveFacts` near `:482` consults established identity, exact identity,
declared identity and contextual values before its status conversion at `:514`.
An overlay that supports GetEntity but omits ResolveAlias, Entities,
attestations or compact-index reads is not semantically equivalent. The status
fixture establishes an actual relation/value change, not a theoretical risk.

`queue/queue.go:396` calls the extractor anew before every ApplyWith attempt.
Its failure path at `:453` retains pending input and retry fields, not res.
The permanent error predicate at `:513` does not include an admission refusal.
A new generic refusal error retries indefinitely with fresh provider calls;
classifying it as permanent parks the complete episode, withholding its valid
facts. Pretending it is ErrAliasClaimed would misstate the cause. Therefore
whole-episode refusal is neither a complete ingestion solution nor a safe
one-line containment patch. The synthetic 1/1 withheld result is a fixture
measurement, not an estimated production percentage.

## Smallest viable architecture and required invariants

Use the existing Badger transaction as the provisional resolution view and
add an extraction-scoped journal and final admission phase. Do not call
DeleteEntity as undo. This adds fewer semantic moving parts than a separate
entity overlay and allows valid assertions plus unresolved structured evidence
to commit together. This is a design selection, not proof of an implementation.

1. Before any alias admission for a proposed new identity, establish whether
   its en: key existed at the transaction snapshot. Retain this classification
   throughout the transaction; never promote a preexisting hollow into the
   newly-created set. For a newly absent identity, old orphan attestation
   episode IDs have zero admission authority. Preserve their exact bytes, but
   count only evidence belonging to the current provisional incarnation.
   Apply this before both declarations and endpoint-driven creation, including
   subsequent declarations in the same transaction. Existing established
   identities keep the reviewed existing admission semantics.
2. The journal captures first before-image/existence and final intended bytes
   for each actually touched entity, claim and attestation key, plus the
   originating identity operation. Preserve unknown fields as bytes. At
   finalization restore old claims/evidence exactly and remove only newly
   introduced routing state belonging to unsupported provisional identities.
   Do not scan-delete a key family. Conflicting ownership/dependency in the
   journal is an explicit refusal, never a guessed owner. Previously orphaned
   claims remain disclosed preexisting defects; retaining them is not cleanup.
3. Record each parsed declaration occurrence, including original aliases,
   both description alternatives, TypeFallback, type, exact timestamps and
   source episode link, as well as the final materialized entity bytes and
   admission/attestation decisions. Store exact before/after attestation lists
   for displaced provisional evidence; rejected aliases must not disappear
   because they were absent from Entity.Aliases. Stubs carry endpoint mention
   and origin kind. This is structured extraction evidence, not source
   transcript or a new fact. Keeping the final entity alone is insufficient.
4. Determine support from every authoritative current or historical f: record
   touching either endpoint, including old dangling endpoints newly healed.
   Use the same transaction snapshot and its staged facts; do not rely only
   on adj:. AllFacts at store/store.go:891 uses the transactional view and
   includes invalidated facts. A full scan is a simple correctness prototype;
   quantify its throughput/memory cost before integration. Do not add a second
   mutable support index without its own consistency proof.
5. Finalize after all identity/fact/value decisions. Supported nodes and all
   real assertions retain their existing effects; unsupported new nodes become
   immutable non-routing evidence, with neither en: nor newly created al:/att:
   authority. Do not automatically promote old observations on slug, alias,
   type compatibility or fact count. Existing orphan attestations remain inert
   for future absent-identity admissions under invariant 1.
6. Finalize the event plan to contain only committed graph objects and real
   fact changes. Unsupported provisional put/delete pairs publish no graph
   events. An error after evidence staging or any failed postcondition must
   roll back every raw key and publish zero callbacks. Keep cache release and
   do no further identity resolution after finalization.
7. Ordinary episode reapply is still a no-op. Force with identical evidence
   must not multiply it; Force with changed metadata must preserve a distinct
   immutable version and disclose the conflict. Use full length-framed
   identity/occurrence/episode data and compare complete bytes on collision,
   rather than trusting slug or digest. Do not let an evidence-key collision
   overwrite useful metadata. Leave public Stats comparable; report evidence
   counts through a separate explicit status surface with clear semantics.
8. Add inspection and audit output that finds unresolved evidence by exact
   provenance and spellings without using it to route normal recall. Graph
   hollows and unresolved observations must both appear in grading. An
   accepted episode with such evidence must expose its unresolved count; the
   queue's existing +entities log alone is misleading. No final goal PASS is
   earned by moving metadata outside the graph. Explicit resolution/promotion
   can be a separately reviewed operation; no automatic semantic matching is
   needed for this bounded prevention prototype.

The raw journal is necessary in addition to the semantic support set: restoring
only en:/al: would miss att: provenance; counting old orphan attestations would
already contaminate supported nodes. These are separate invariants, and neither
replaces the other.

## Compatibility and an explicit contract conflict

Keep numeric SchemaVersion 1 for this additive experiment. The backup fixture
proves arbitrary opaque bytes survive current source backup/restore and writes.
I independently inspected `git show a078240^:internal/memory/store/store.go`:
its ensureSchema at lines 263 onward calls DropAll when the disk version is
nonzero and different from its expected version 1. A numeric bump therefore
risks destroying the store when a retained pre-refusal writer opens it.
This source inspection is not an execution or hash verification of the exact
retained host binaries. The selected additive format still requires exact
retained-artifact fixture testing before deployment. An old writer preserving
opaque bytes does not mean it preserves prevention: its Apply still creates
hollows, and it has no new orphan-evidence isolation. A rollback boundary must
report prevention suspended and retain the evidence; it cannot claim success.

`resolve/resolve_test.go:808` TestApply_EmptySlugSkipped positively requires a
committed concept stub, one persisted entity, zero facts and success from
Apply. A universal new Apply contract of zero newly hollow entities cannot
also satisfy those exact assertions for the identical fixture. This is a
direct logical conflict, not a flaky test or an implementation problem.
Do not conceal it by editing expected values, faking Counts/GetEntity, using
test-only flags, or silently retaining a production bypass.

The next allowed step can be concrete without settling that production
contract: build the private prototype behind an explicitly separate internal
entry point, retain the old test unchanged as baseline evidence, and add
disproof tests for the new entry point. Likewise a journal-only primitive can
be developed with no behavioral change. Integration must explicitly adopt the
new admission contract and account for every production caller (queue.ApplyWith
and daemon.MemoryApply/Apply), then reconcile the legacy test openly. Merely
adding an option while leaving a normal writer unguarded is not prevention.
This report grants no live apply, schema change or default behavior change.

## Bounded verdict and remaining proof

**GO** for the private transaction-journal plus new-identity attestation
isolation prototype described above, keeping a separate entry point and all
existing assertions intact. This is useful authorized design work with no need
to stop for a speculative architecture preference.

**NO-GO** for integrating simple final DeleteEntity, for a materialized-Entity
only evidence format, for whole-episode refusal presented as completion, or for
an overlay lacking the entire Store read view. **NO-GO** for deployment until a
fresh grader proves the explicit new contract, raw-key preservation including
failure injection, alias-incarnation isolation, valid mixed-episode ingestion,
historical/dangling support, duplicate/Force handling, observer/index behavior,
visible unresolved evidence, backup/retained-writer behavior and bounded cost.

I did not construct or grade a production implementation, change source
expectations, run the full repository suite, restore the live store, execute
providers, or inspect raw live facts. Existing final-goal defects, old hollow
identities, status semantics, fifty held-out questions, actual recall/p95 and
the two complete grading rounds remain outside this verdict. No old live
hollow is authorized for retirement by this admission design.
