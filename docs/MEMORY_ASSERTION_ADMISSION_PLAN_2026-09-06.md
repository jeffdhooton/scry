# Complete assertion planning and admission contract — corrected direction v2

2026-09-06. Root implementation direction, not deployed behavior or approval.
This replaces the rejected comparator-only bridge, not the preservation goal.
The delivery outcome is the normal Apply path using one owned complete plan,
durable per-occurrence dispositions and verified materialization. A standalone
comparator, codec, read-only report or another uncalled helper is not delivery.

This successor incorporates independent review a3c2508199b853153ae233bc690cc50191779d7df160d72abfa35087aeed73f5.
The original draft508a70c2 remains immutable. The corrections below are root's
explicit implementation requirements, not a claim that the reviewer tested code.

## Evidence and constraints

Read the active objective at
/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md.
Read the complete semantic disproof86773de0218016c370ea37570f5309500503395dc384f1197cfcc0e1db17b26e
and experiment68987a69f23688becddc598ecb328640d27644fca173638039714e297b63c670,
archived in docs/memory-repairs/assertion-bridge-*. Preserve its unchanged tests
f265ef5bbefe0dd24e5b4762943f8117d5bc79d236e8663f027503e6daae3dd1 and root tests
03d60f30e75aca028acb412cef97aa8e286505ab66ae557758b48a268c7b5d5a. Some are
characterizations of failure, not success assertions; retain and supplement them.

Controlling related reviews: ControllerV3 review2521939a, assertion-identity
review5f3cc109, identity-mutation review94e6e51f. Their complete paths and hashes
are in the existing handoff/audit and archived memory-repairs files. Delayed-birth
and ordered-overlay contracts remain responsible for original identity discovery,
not temporal authority. Fifth identity overlay43a61c97 is still under independent
review; none of its private source is approved by this draft.

No new lexical dictionary, word threshold, type inference, provider, relation or
automatic stored-identity owner choice. No silent sentence/literal/start loss,
timestamp nudge, backdate relocation or mutation of the old assertion's content.
No new on-disk FA address or recovery in this stage. Legacy same-address conflicts
are durably deferred until an independently reviewed exact FactRef/sibling format
and all-reader/writer transition can preserve them. Deferred original input is not
an accepted fact, an ingestion-success certificate or recovered lost history.

## One owned program, not caller-selected authority

The fixed store-owned entry receives the complete canonical original input revision,
actual episode bytes and ordinary execution options. It clones/validates input
before waiting, acquires the genuine graph coordinator before its transaction,
owns baseline identity/FA inventories, original observations and both write ledgers,
then executes the identity overlay and the full assertion plan in that same scope.
Do not call the public overlay wrapper inside another transaction/lock or accept a
caller-made overlay report. Factor its fixed internal program for composition,
with the existing read-only wrapper retained as a characterization entry.

Caller supplies no support set, birth handle, chosen owners, finalizer, approved
mutations or synthetic raw witnesses. Production resolve.Apply becomes a thin
adapter to this fixed entry; old Phase A/Phase B write loops are not mixed with it.
Canonical relation and lexical policy move behind the shared neutral policy package
before activation; copied implementations cannot remain two production authorities.

Capture every original declaration/assertion ordinal, raw ValidFrom string,
finite JSON confidence and complete nil/present SupRef before any mapping. The
existing input codec is the finite canonical-input boundary, not a NaN-payload
envelope: nonfinite confidence or invalid episode time refuses safely before this
entry and leaves the existing queued original intact. Do not claim a local durable
disposition for unencodable caller values. Finite but semantically invalid confidence
can receive a local assertion deferral from the original canonical input. All primary
discoveries precede all hint resolution. Keep original endpoint side/role after
inverse mapping and value-source reversal. Rejected primaries can still have
intermediate identity proposals visible to later resolution, but visibility is not
support or permission to materialize them. Full input/observations persist under
the existing strict codecs; reports own bytes and expose no active capability.

## Exact assertions, actual addresses and metadata

Assertion identity is the full tuple: resolved source, canonical relation, target
discriminant with exact destination identity OR complete stored literal, exact
stored raw relation, exact statement, and supported UTC start instant. Hashes are
indexes only; compare the entire tuple. Preserve the complete original mention and
unparsed fields in observations even where existing policy trims a stored literal
or normalizes a raw relation. Confidence, episode provenance and InvalidAt are not
identity fields. Keep the actual legacy raw address and full raw bytes separately.

For all relations, current and historical alike:

- An exact stored repeat preserves its address, start, statement, literal, raw
  relation and existing InvalidAt. It can union actual episode provenance and take
  maximum confidence. It does not reactivate history or trigger a new automatic
  exclusive transition merely because it was observed again.
- An exact repeat among new occurrences produces one materialization with max
  confidence and deduplicated provenance, while retaining every original ordinal
  and its own hint/observation dependencies. Never overwrite .95 with .2.
- A distinct assertion at a vacant legacy address can be proposed independently.
  Distinct earlier/later starts remain separate. Different assertions sharing an
  occupied or mutually proposed legacy address defer the whole overlapping-address
  component; no first/last writer wins or manufactured sibling timestamp.
- Equal timezone-adjusted instants compare equal. Zero/out-of-UnixNano-range times
  and invalid confidence cannot supply valid assertion identities.

After final inverse/value routing and identity resolution, an edge with nonempty
destination and Src==Dst is a typed local self-loop deferral. Its complete primary
and hint observations remain durable, but it contributes no new support and produces
no invalidation, metadata or vote effects dependent on that assertion. Unrelated
facts survive. This is endpoint equality, not lexical inference or a relation rewrite.
Existing self-loop rows remain untouched; incoming exact restatements of them also
defer rather than reactivating, extending or automatically cleaning them. Previously
terminal occurrence results can still be carried forward unchanged on retry. Neither
that carry-forward nor preservation of old loops certifies a clean graph.

Before any existing FA metadata or interval rewrite, decode and key-bind the full
raw row and require that the supported Fact representation round-trips exactly.
Unsupported extensions, duplicate/case-ambiguous fields, lossy numbers/strings or
other noncanonical representations cannot be silently reserialized. A valid but
unsupported representation defers its dependent component; exact absence and full
before-image are dependencies. Structurally malformed baseline FA inventory still
refuses the transaction under the existing strict inventory contract. This is not
an independent arbitrary-raw-writer protection promise.

Every rewrite retains unrelated supported fields and full provenance. New episode
references require the actual immutable episode proof staged in the same owned
transaction; old provenance references require their actual valid EP rows. Unknown
raw data stays untouched when no approved rewrite consumes it. The direct public
InvalidateFact/DeleteFact and relocation consumers must join the eventual writer
floor; the new Apply route alone is not an all-writer certificate.

## Explicit date and exclusivity policy

Absent/blank ValidFrom uses the episode instant, as before. A nonempty malformed
explicit date, or a parsed instant outside the supported range, is a typed local
assertion deferral. Preserve its raw date and observations; do not guess the old
assertion's identity from the fallback instant. Its unrelated components still
commit. This intentionally replaces the old malformed-date fallback, but does not
repeat the bridge's whole-episode refusal. Independent review must measure both
the local refusal and the retained unrelated utility before test migration.

For exclusive relations, distinguish observation time from effective state time.
Automatic transitions use actual assertion ValidFrom, not arrival order or the
later episode instant when an explicit start was supplied. No new instant is
invented. Full typed targets compare exactly; AttrDst/Normalize is not equality.
Multiple separate assertions about the same target may overlap without constituting
contradictory exclusive targets. Never merge their sentences to obtain that property.

Plan a touched (source, canonical exclusive relation) cohort as a whole, independent
of input ordering. Use every relevant current and closed baseline interval and all
candidate new assertions. Exact stored repeats contribute only their existing
interval, not a new state event. A genuinely new assertion starts at its supplied
effective start; its proposed end is the earliest strictly later different-target
start that is independently present in the retained baseline/new timeline. When
there is no such transition it remains current, subject to the checks below.

A baseline current assertion may close only at an actual strictly later retained
new different-target start; it must not be closed by merely re-observing history.
Keep every already-closed baseline interval unchanged. Equal starts with different
targets have no justified winner: defer new assertions/effects in that cohort.
Do not turn this into an earliest/latest/confidence/ordinal tie-break. A new earlier
state can be recorded as historical ending at a known later state start; observing
it later cannot retire that newer state.

Before accepting the cohort, sweep its proposed half-open intervals [start,end),
including all baseline closed history, to prove no newly introduced overlap between
different full targets and no nonpositive interval. An overlap with immutable old
history that cannot be resolved without changing that history defers the cohort.
An inherited inconsistent cohort is not automatically repaired by an unrelated new
sentence: preserve its baseline defect and defer new state effects unless a reviewed
repair separately establishes intervals. Metadata-only exact repeats can remain
eligible if they introduce no state effects and their own hint dependencies pass.
This conservative limitation must be explicit in outcomes, not hidden as success.

## SupRef means eligible unique target, not first stored triple

Hints have no own start; their event time is the containing episode's OccurredAt.
After all primary identity resolutions and preliminary exact assertion proposals,
collect the full target set from immutable baseline PLUS all proposed primary rows.
The preliminary interval is the interval computed from that frozen baseline and
proposed exclusive timeline, before SupRef effects. Its transition suppliers are
explicit dependencies. No row is mutated during either planning phase. Resolve/validate both hint
identity routes; dangling old FA endpoints cannot authorize a new identity.

Match canonical relation and complete typed target. For fallback relations also
match the full normalized raw relation; for canonical relations a triple-only hint
does not distinguish different stored raw verbs or statements, so they all remain
candidates. Full assertion identity deduplicates repeated occurrences, not distinct
rows/sentences. An empty/invalid normalized hint relation is an explicit no-effect
hint, retaining its original observation; an unresolved required route defers its
primary-plus-hint component rather than guessing a natural owner.

Validate candidate raw records and endpoints first, then filter temporal eligibility
using the preliminary interval: start < hint event and (end absent OR event < end).
Thus future/equal-start candidates cannot block uniqueness; an already closed row
whose interval contains the event can still establish target ambiguity, but may
not be rewritten under the closed-history rule. If its existing end equals the
event, that alone is not evidence that this occurrence previously ended it. An
already-ended replay result must carry the exact prior committed target/effect and
predecessor outcome; otherwise use ordinary current candidate eligibility.

Zero eligible candidates gives an explicit no-target/no-effect disposition. One
eligible candidate gives an exact address/full-tuple target; if already historical
and would need a changed end, defer this dependent component. More than one defers
all assertions/hints whose candidate sets overlap; no arbitrary winner. Candidate
sets are frozen before hint effects: repeated hints cannot shrink the set, turn
ambiguity into uniqueness, or successively invalidate additional assertions.
The same primary's exact restatement and its hint are planned together, so a later
metadata union cannot undo that hint's interval effect. A self-target with a newly
supplied start equal to event is ineligible, not a zero-length fact.

Combine compatible invalidation requests using the earliest actual supported end,
never moving a closed baseline end. Do not let an exclusive effect derived from a
new assertion survive when the assertion's own hint component is deferred.

## Components, closure and a single materialization

Initially bind each primary to its complete optional hint. Union dependencies for
overlapping exact FA addresses, same new assertion materialization, touched exclusive
cohorts with state effects, actual SupRef eligible candidate/target sets, and identity/
alias proposals required by those routes. Retain the full frozen matching universe
as evidence, distinct from temporal eligibility and component edges. A proven future
start remains ineligible regardless of its occurrence's later acceptance, so that
occurrence's unrelated failed hint alone must not withhold the eligible unique target.
Connect every supplier whose exclusion could change eligibility or a proposed end,
including exclusive transitions and raw/identity proof dependencies. Excluding a
connected proposal removes its dependent effect; it never shrinks ambiguity into
a winner. Mere use of the same unchanged entity does not connect unrelated relations.

Mark unsupported/conflicting components deferred before ANY identity metadata or
FA write. Preserve detailed original per-occurrence outcomes; non-assertions and
optional alias rejections are not fabricated assertion materializations. A recognized
old identity may keep independently valid descriptive metadata without a new fact;
new identities require final retained valid FA support and cannot survive on a name,
description, old attestation count, candidate count or episode marker alone.

Close support/ownership over retained components and their actually consumed identity
proposals. Validate complete affected literal/natural/index relationship closure,
full raw owner controls and explicit ownership dispositions. Old unresolved defects
may only remain as exactly unchanged defects under the reviewed identity contract;
support never grants an alias transfer or cross-type merge. A failed required owner
or unsupported candidate removes its dependent component and effects. Recompute
support until stable by monotonic removal only: no rerouting, rediscovery, new
candidate selection or reduction of an ambiguous hint set into a winner. Optional
alias proposals may be rejected without withholding independent useful assertions.

Persist only the resulting approved identity metadata, new supported births and
current-generation votes, exact FA insertions/restatement unions/invalidation ends,
actual episode, complete original observations/input, disposition records and
selected current outcome head in one transaction. Unsupported candidates are never
temporarily staged; their outcome materialization is nil. Finalizer verifies exact
identity and FA ledgers against owned before-images and final inventories, no missing
or added unplanned rows, full historical preservation except explicitly allowed
current-to-closed transitions, provenance closure, alias ownership and no unsupported
new identity. Include exact adj mirror before-images and approved deltas alongside
FA writes; neither ordinary PutFact nor the new writer may silently repair an old
stale/missing mirror outside the approved delta. Buffer events until commit; failure
yields zero committed Stats/events.

The existing v1 identity outcome is insufficient for this result: it requires a
deferred endpoint for assertion deferral and forbids assertion materializations.
Preserve those v1 bytes and bounded readers. In the composed program, version the
occurrence-result envelope or bind it to an immutable exact planner result. Record
assertion disposition/reason independently of truthful endpoint-resolution states,
full input revision/ordinal, actual FA address/full tuple and materialization or
absence, preliminary/final intervals, complete frozen hint matching/eligible sets,
exact selected target/effect or explicit no-effect reason, and supplier/predecessor
dependencies. Optional alias rejection and non-assertion remain distinct from FA
success or unresolved identity. The fixed owner derives and verifies these records;
they are neither caller-selected permission nor reconstructed guesses after replay.

Force first pins the exact original input revision and previous complete selected
result/head. Within that revision, carry forward already committed occurrences and
their exact assertion/materialization and hint-effect references. Do not execute
their hints, exclusivity transitions, metadata or votes again because a different
occurrence remains deferred. Only retryable deferred components replan against
revalidated current dependencies. Committed facts may be baseline dependencies for
newly retained work without reopening the old occurrence's effect program.

If an old exact target moved under a reviewed repair, follow explicit recorded repair
lineage or report stale/conflicting evidence. A matching triple or end timestamp
cannot identify a prior effect. Same-input retry with unchanged results reuses the
immutable records and exact head, with no extra votes/events. An actual disposition
change appends/selects a complete successor preserving terminal occurrences. HasEpisode
never implies full success; public partial/deferred status remains measurable.

A changed extraction revision under the same EP is not implicit authority to undo
or rerun accepted facts. At this first activation boundary, preserve the new input
revision durably but refuse its selection with explicit revision-conflict if the old
selected revision has any accepted occurrences. Leave the old selected head and all
accepted effects intact. If the old selected revision accepted none, the new complete
revision may be planned normally. A later broader revision-mapping operation needs
its own explicit preservation/authority contract; do not pretend this conservative
conflict is successful changed-revision ingestion. This is an implementation scope
boundary, not permission to waive any final full-goal coverage requirement.

All mutations operate under the same cooperative writer floor; adoption must not
become live until every writer and generic metadata path is lifecycle-aware.
Unadopted legacy mode cannot be marketed as this prevention certificate.

## Implementation and proof gates

1. Independently disprove these specific temporal, candidate-set, raw-preservation
   and dependency rules against actual current source and retained semantic cases.
   Correct the contract rather than layer another helper on an unresolved policy.
2. Implement the composed plan privately, preserving all earlier failed exports and
   tests. Exercise normal Apply through a fixed owned adapter on synthetic current,
   historical, conflicting, out-of-order and partial-utility inputs. No supplied
   test edits to conceal behavior; reconcile incompatible old merge/date assertions
   explicitly only after independent acceptance of replacement requirements.
3. Obtain fresh exact-source whole-program disproof, full no-CGO suite, and restored
   actual-backup conservation/adoption/partial-replay tests before integration.
   Demonstrate actual committed rows, outcomes, identity support and events—not just
   planner output, Stats changes or proposed mutations. Keep earlier code-only
   overlay/codec grades separate from this complete write-path result.
4. Finish all-writer enforcement, protected namespace/schema floor and rollback
   incompatibility proof before live adoption or deployment. No live cleanup/recovery
   follows merely from this planner's success. Complete the original benchmark,
   alias/Hermes/hygiene/recall/sweep and two full-goal-round bars unchanged.

Required counterexamples include both orders of equal/different explicit-start
exclusive inputs; later observation of an older effective state; same-target
distinct sentences; normalized-equal full literals; occupied legacy addresses;
current/historical/timezone-equivalent repeats; confidence/provenance dedup; 1971/
2002 candidates at a 1980 hint; canonical and fallback ambiguous hints; repeated,
self and cyclic hints; future/closed targets; optional/required identity failures;
raw extensions on EVERY rewrite route; malformed-date unrelated utility; duplicate
occurrences; unsupported discovery later used by a retained assertion; Force partial
retry and reopen; transaction conflict and zero leaked events or sensitive errors.

## Controlling independent implementation clarifications

The following complete addendum is part of this implementation contract. Its explicit operational interpretations control any ambiguity above. It grants design direction only; implement and independently test the whole owned program before any promotion.

# Corrected assertion contract: bounded follow-up

2026-09-06. Reviewed `/tmp/scry-complete-fact-plan-sep06.VK8j3B/COMPLETE_FACT_PLAN_V2.md`, SHA-256 `cf278586db370d736b4522da155900c9fda50fac406e1b138f94f9650e754f9d`, against original `508a70c2fbce2d1e9b3e8062577469d5d6144c4d10f53351aa00a3ae4e05bdd3` and my report `a3c2508199b853153ae233bc690cc50191779d7df160d72abfa35087aeed73f5`.

**The three findings and associated precision changes are faithfully incorporated. I found no remaining contradiction in the changed sections under the explicit interpretations below. Proceed with the composed private implementation and its independent whole-program tests; this is conditional design direction, not code, overlay, production or full-goal approval.** No additional helper is a prerequisite.

## Findings closed at the design level

1. Same-revision partial Force now carries committed occurrences and exact hint effects forward, while only deferred components replan. In the X1/day-1, E/day-10, later X2/day-5 timeline, retrying unrelated R cannot execute H again, so H cannot acquire X2. Already-ended replay requires the prior target/effect and predecessor; timestamp equality is correctly insufficient. Unchanged outcomes reuse the head, votes and events.
2. The new occurrence-result contract acknowledges v1's limitations and records assertion disposition independently of endpoint states, plus exact FA/interval/candidate/effect/revision evidence. This removes the need to invent an unresolved identity to encode an FA conflict. Existing v1 bytes/readers remain preserved. Actual schema and fixed-owner verification remain implementation work.
3. Both newly materialized resolved self-loops and incoming exact restatements of existing self-loops now defer locally, without assertion-dependent metadata, votes, support or invalidations. Old rows remain untouched. Carrying a previously terminal result forward performs no fresh materialization and explicitly is not a clean-graph certificate. This conservative existing-loop exception is now a stated policy, not silent cleanup or reactivation.

The preliminary interval now explicitly uses the frozen baseline/proposed exclusive timeline before SupRef effects and records transition suppliers. The matching universe, eligible targets and dependency edges are distinct. A proven future candidate's unrelated failed hint cannot alone withhold the unique eligible target; exclusions that can change eligibility/end remain dependencies. Exact adj before-images and approved deltas address unintended mirror repair. These changes match the earlier review's precision requests.

The finite canonical JSON boundary is also accurate. Nonfinite confidences and invalid episode instants refuse before the owned entry, preserving an existing queued original rather than pretending to create a local durable disposition that the codec cannot encode. Finite out-of-policy confidences can still receive truthful local dispositions. This is a narrower, explicit input contract; the new code must not manufacture a missing queue item for a direct caller or report an unpersisted value as durably captured.

## Changed-revision boundary

The conservative rule is coherent: once selected revision R1 contains any accepted occurrence, a differing extraction R2 can be retained as input but cannot replace R1's selected head or execute graph effects. R1's accepted facts/hints are not rerun or undone. The refusal is public revision conflict, not successful extraction ingestion. An R1 with no accepted occurrences may be superseded by a newly planned complete R2 while retaining R1's immutable observations/results. This is a bounded first-activation policy; it does not waive later full-goal coverage.

Three implementation consequences are essential to that reading:

- **Refusing selection is not aborting preservation.** On R1-with-acceptance/R2-conflict, commit only the durable new input/associated conflict evidence while retaining the exact old EP/head and all graph effects. Returning revision-conflict from inside the existing AtomicWrite callback would roll back that new input and fail the stated durability requirement. Return a committed typed selection-conflict result outside the successful transaction, or an equivalent explicitly distinguished API outcome. Actual storage failure remains rollback, not a persisted conflict. Repeating this request deduplicates the retained revision/evidence and never advances the selected head.
- **Any accepted occurrence is derived from the complete selected result.** It is not a caller flag, current-FA count or an inference from HasEpisode. Include carried terminal occurrences and any committed occurrence whose accepted effect is a restatement union, invalidation, identity metadata or vote rather than a new FA insertion. A component with accepted effects cannot be relabeled all-deferred to unlock revision replacement. Baseline identity metadata already permitted independently of assertions stays preserved; accepting a new revision does not authorize reversing it. If a prior head/lineage cannot establish the no-acceptance condition, do not assume it.
- **Same EP means the pinned immutable episode proof, not merely the same text ID.** A changed extraction summary belongs to the new input revision; it does not overwrite the existing EP Summary/IngestedAt or its provenance identity. A changed source/time/body witness must not pass through the old-none-accepted exception as if it were only a different extraction. Validate/pin actual EP bytes under the owner before preserving/selecting revisions, as the unchanged contract already requires.

These are direct operational readings of v2's durable retention, complete-result, immutable-episode and no-rerun requirements. They do not require another design/helper cycle. The resulting public counts must distinguish the selected revision's partial/deferred work from a durably retained but unselected conflicting revision; the latter must not disappear merely because the selected head is complete.

## Bounded evidence and remaining gate

This follow-up read the exact diff and relevant v2 sections, inspected the unchanged baseline AtomicWrite/Apply and EP shape/writer semantics as needed, and authored only this addendum with apply_patch inside my existing private export. No source/test edits, new tests, actual stores, backups, replicas, providers, shared files, configuration, agents or external writes occurred. Prior tests and logs remain unchanged; they are characterizations, not v2 implementation evidence.

Required new implementation fixtures should include: partial R1 accepts H/deferred R, later X2, Force R1 carries H unchanged; differing R2 is durable/unselected and leaves old head/EP/FA bytes intact; repeat that conflict then reopen; zero-accepted R1 allows complete R2 selection without deleting R1 history; an accepted metadata/invalidation-only occurrence blocks the no-acceptance shortcut; finite-invalid confidence local utility; nonfinite pre-entry refusal; self-loop plus unrelated retained assertion and exact old-loop restatement deferral. Original exact-source whole-program, no-CGO, actual-backup/all-writer/adoption/rollback and full-goal gates remain unchanged.

The addendum hash is returned separately after writing.
