# Register exact candidate births before body mutations

2026-09-06. PRIVATE DESIGN, not implementation or production approval. Baseline
4f1d1be plus reviewed serialized coordinator source8093c39a/f51c3dec4/396389b0,
contract91071f65 and code reviewcc5b9697. Complete canonical inputb69e6daf,
identity mutation ledgere43d2b90 and fact mutation ledger2040e19a are available.
ControllerV3, identity mutation design review94e6e51f and current episode review
a9722fcf remain mandatory constraints, not closed by registration.

## Why this next boundary exists

Actual write accounting cannot tell which new identity was intentionally born in
this episode. Registration must precede the first alias proposal/PutEntity, bind
the exact parsed occurrence, and remain the same identity through later mentions.
An intermediate en: row must not turn a provisional birth into a legacy identity
or let old att: votes select it. Registration itself grants none of those powers.

## Proposed private root entry and owned state

Use a private registration harness that delegates to the reviewed serialized owner.
Its inputs are full canonical parsed revision key/raw and private body/finalizer
test seams. Before invoking the caller's BODY, the wrapper decodes and owns the
revision, creates its OWN complete identity mutation ledger and fact mutation
ledger, then constructs the registry bound to that same owner/facade. No caller
baseline, support flags, existing registry or prior writer object is accepted.
The wrapper establishes capture-before-body; a public method called arbitrarily
late in an already-mutated transaction is not an acceptable substitute.

The public production API does not yet exist. The harness callback is a testing
seam only. Eventual production admission must fix initialization and finalization
internally and must not expose caller-chosen finalizers. Existing private harnesses
remain unchanged and retain their counterexamples.

The full parsed revision is available before normal resolver writes, but the new
ep: row is currently written LAST. Therefore registration performs only pure
canonical input/observation matching during body, not a premature raw-EP proof or
early PutEpisode that would trigger Apply's idempotency skip. Final persistence
must later validate actual ep: identity/time through the reviewed provenance
writer before a result may commit. This registry stores no source text/spans.

## Candidate registration

The new-birth request identifies exactly ONE primary parsed observation (full key
and raw bytes). Match it against the owned revision using full canonical matcher.
Only declaration name or endpoint primary src/dst is a birth origin; supersedes
references are dependencies, never birth origins. Derive the complete identityBirth
tuple internally: episode/time from revision; origin/ordinal from observation;
name from declaration Name or the selected original fact side; slug=Slugify(name).
Validate with identityBirthRecord. No supplied slug, creation time, generation ID
or caller-chosen stable identity tuple is accepted.

For a FIRST registration of that derived slug:

- Require baseline en: absent, current en: still absent, and baseline/current ig:,
  il:, il-consumed: selection/consumption keys absent. No reuse of a legacy or
  previously selected identity. Include rs:slug and rt: normalized birth name /
  normalized slug presence in conservative refusal; opaque occupied bytes refuse.
- Require zero baseline raw fact references, current OR historical, through the
  wrapper's owned complete fact inventory. If references exist, return a typed
  deferred-reference result preserving exact observation identity, not an empty
  slug, fabricated value, new entity or generic retry error. Other valid inputs
  must remain processable by the future complete controller.
- A preexisting entity is an existing-identity path, not a birth. Return explicit
  not-a-new-birth/refusal, never silently adopt it or construct a generation.
- Registration does not inspect name plausibility or select alias ownership; keep
  resolver artifact/established-identity/type/context value guards separate.
  Old unlisted alias keys pointing to the absent slug remain measured by the full
  relationship baseline. They confer no authority. The eventual closure may defer
  or refuse such reuse; zero fact references alone is not alias-relationship proof.

Store the first exact canonical birth record and the first observation identity.
Every operation owns caller buffers. Duplicate exact registration is idempotent
and returns the same owner-bound handle. A different occurrence/name mapping to an
already registered slug must NOT silently create another birth or replace the
first tuple. It must use an explicit mention-link operation on the existing handle,
or refuse the attempted second creation. No store-scale name-based owner inference.

## Subsequent mentions and freeze

A private mention-link operation accepts only a genuine registry-issued handle and
an exact revision-matching observation, with an explicit role when it refers to a
Supersedes field. Preserve all distinct declaration/primary/hint occurrence links,
including repeated names, conflicting descriptions, aliases, TypeFallback and
original unparsed ValidFrom. Deduplicate only exact observation-key/role pairs.
The operation is MECHANICAL attribution, not permission to bind a spelling to that
identity. The fixed resolver/ownership policy must validate those mappings before
using them for routing, support or outcome resolution annotations.

Keep first-birth origin distinct from later mention roles; a hint cannot rewrite
it. Scope/owner/facade/phase checks apply to every operation and ignored failure
poisons the owner. No registry method writes en:/al:/att:/ig:/iga:/io:/ep:/fa: or
emits events. B votes remain a separate unintegrated mechanism and are not imported
or certified by this unit.

Freeze once during finalizing into a fully owned deterministic registration report:
creation-order entries, full canonical births and full first/additional observation
identities, plus explicit deferred registration observations. No report field is
called supported, committed, completed or current. Verify both owned actual ledgers
before reporting the end-of-body accounting; unmatched en: creations discovered in
their actual history must be exposed/refused by the eventual fixed policy, not
silently added to the registration inventory. The next unit must state whether its
freeze is merely registration accounting or additionally enforces exact creation
coverage; do not claim coverage from a caller's completeness promise.

## Force and full-controller bar still open

This report describes births registered in ONE transaction. On identical Force, an
already selected identity is no longer newly absent. Do not drop its historical
birth from the complete current episode result merely because this registry has
no new creation. That requires exact prior selected result/input proof and stable
birth inventory reuse, as current episode reviewa9722fcf requires. This registry
does not implement that replay merge or permit automatic resurrection of consumed
identities. Deferred decisions are outstanding ingestion defects, not success.

Support selection still needs actual admitted assertion/dependency verification
BEFORE both resolver fact phases, including Supersedes and exclusive invalidation.
Undo needs full approved relationship closure, not just registry membership; B
materialization and post-verify writes need exact finalizer accounting. All-writer
lifecycle enforcement and useful bounded result/status/orient inspection remain.
No production routing, live adoption or cleanup follows from a registry verdict.

## Design disproof request

Is this a useful safe finite next registration boundary, or does it leave a proven
ambiguity that should be resolved before implementation? Especially test derivation
under relation flips, hint-only origins, same-slug different occurrences, capture
timing, first registration after raw alias/entity writes, stable Force inventory,
and the distinction between mechanical mention links and approved identity use.
Recommend the smallest coherent implementation scope. Do not certify the complete
controller from an unimplemented design. Keep all prior failed regressions intact.
