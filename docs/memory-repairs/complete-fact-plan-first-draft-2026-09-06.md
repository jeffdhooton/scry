# Complete assertion planning and admission contract — draft for disproof

2026-09-06. Root implementation direction, not deployed behavior or approval.
This replaces the rejected comparator-only bridge, not the preservation goal.
The delivery outcome is the normal Apply path using one owned complete plan,
durable per-occurrence dispositions and verified materialization. A standalone
comparator, codec, read-only report or another uncalled helper is not delivery.

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
confidence bits and complete nil/present SupRef before any mapping. All primary
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
collect the full target set from immutable baseline PLUS all proposed primary rows,
before applying any hint or exclusive invalidation. Resolve/validate both hint
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
event, retain a separately recorded already-ended no-effect result on exact replay.

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
cohorts with state effects, actual SupRef candidate/target sets, and identity/alias
proposals required by those routes. Include all potential candidates before filtering
accepted effects; excluding one proposal must not silently reinterpret another hint.
Mere use of the same unchanged existing entity does not connect unrelated relations.

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
new identity. Buffer events until commit; failure yields zero committed Stats/events.

Force replay appends immutable observation/outcome history and selects a complete
current result with exact expected-head checks; it does not rewrite history or let
HasEpisode imply full success. Partial/deferred status is public and measurable.
Repeated same-input completed execution is an exact no-op; partial retry may produce
a new explicit result only from revalidated full dependencies. All mutations operate
under the same cooperative writer floor; adoption must not become live until every
writer and generic metadata path is lifecycle-aware. Unadopted legacy mode cannot
be marketed as this prevention certificate.

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
