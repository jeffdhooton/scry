# Controller V3 independent bounded design review

2026-09-06. **GO for a private, uncalled immutable outcome codec with the subject correction below. NO-GO for production controller integration or legacy adoption on this draft alone.** The B buffer, exact legacy inventory, immutable input/outcome separation and explicit partial-episode direction can compose. That is an architectural judgment, not an implementation PASS. No controller, adoption implementation or full integration proof exists in the reviewed input.

I ran the required session orientation, read the complete active objective, complete exact V3 draft and complete archived V2 independent review, and inspected the relevant baseline resolver/store/observation/journal sources. I exported commit `06bcaf0731dd4cf5f833e8a1e97776693d3ee716` into `/tmp/scry-controller-v3-review-UMV0Ro`. Only this report and the new characterization test were authored there. No shared/frozen source edits, live graph access, provider calls, sweeps, config changes, memory writes, room posts, deployment or other external writes occurred. The two test cases characterize existing resolver semantics, not an unimplemented controller. I did not repeat another grader's raw-reference or B primitive review.

## Concrete characterization and necessary design precision

### Deferred dependencies must be resolved before either fact phase

V3 lines 82–91 correctly require deferring mutations dependent on deferred assertions. The current resolver cannot implement that simply by excluding unresolved primary facts or skipping Supersedes later.

Independent test `TestControllerV3FilteredPrimaryStillInvalidatesDanglingSupersedes` creates Atlas, Borealis and Cygnus, writes an old `Atlas uses Cygnus` fact, then removes only the synthetic Cygnus entity. It applies a retained assertion `Atlas uses Borealis` with Supersedes `Atlas uses Cygnus`, representing a mixed episode after its separate primary Cygnus assertion has been filtered. Existing Apply successfully adds Borealis's assertion AND invalidates the old dangling-target assertion. `resolveSlugOnly` returns the absent natural slug; `applySupersedes` then finds the current raw fact and calls InvalidateFact without an identity existence/anchor check. No endpoint entity was created for Cygnus. Its historical fact remains, but its validity changed.

Independent test `TestControllerV3SupersedesDependencyIsAfterPhaseAMerge` also seeds the Borealis assertion. Phase A updates its confidence and episode provenance before Phase B resolves the dangling Supersedes reference and invalidates the other fact. A future late hint guard that merely returns nil would therefore commit part of an assertion which an outcome might label wholly deferred.

Both tests PASS as baseline characterizations. They prove the dependency boundary a V3 implementation must replace; they do not prove that V3's stated dependency rule is impossible or already violated by an implementation.

Smallest correction: explicitly resolve primary endpoints AND Supersedes endpoints, classify lifecycle failures, and compute the mutation dependency closure before Phase A merges or Phase B invalidates. Preserve exact original occurrence ordinals and complete SupRef. The current SupRef is only a triple, not an original assertion ID or fact-version ID; do not invent an exact input dependency when multiple occurrences/old rows match. Record derived links with their role and all relevant exact observation/fact references, or conservatively defer the ambiguous dependency component. State whether the entire assertion-plus-hint is atomic; that is the simplest first contract. Unrelated components must still commit. Include exclusive invalidation, historical-address preservation and backdated merge DeleteFact/PutFact relocation in the dependency/write inventory.

The explicit all-writer list at V3 lines 176–184 omits InvalidateFact and DeleteFact, and fact-only primary endpoint validation does not validate Supersedes endpoints. Add them to the executable lifecycle/mutation policy. Do not claim raw support counts alone validate the authority of those mutations. A policy for existing dangling targets reached only through exclusive invalidation also needs an explicit decision and test, rather than silently inheriting an old helper's behavior.

### A deferred assertion does not necessarily have a birth

V3 lines 136–139 require each outcome to contain a complete birth identity. Lines 77–80 explicitly forbid a speculative birth for an absent slug with old references. The synthetic Supersedes case is stronger: its primary endpoints are both established and the deferred identity exists only in the hint. There may be no registered new birth in the assertion at all.

This is a schema ambiguity in the design, not a reproduced codec failure. Do not force the preservation codec to manufacture birth/generation authority to record this case. Use a closed subject distinction: a registered-birth outcome contains its complete birth tuple; a deferred-assertion outcome contains exact occurrence keys and typed unresolved endpoint/hint references, with no selected generation or staged entity required. A proposed spelling/slug is descriptive input, never a recognized owner. An established-primary outcome may link exact recognized anchors separately. Model Supersedes source/destination roles explicitly; the present observation side enum only covers primary `src`/`dst`, although it correctly preserves the full SupRef inside the fact payload. Keeping that exact observation payload and adding role annotations in the outcome avoids pretending the observation primitive already supports new origins.

One fact can touch two new births. Link its exact observation keys without counting the fact twice. A dangling/hint-only deferred assertion also needs observation preservation even when no occurrence qualifies under the narrower “touching any new birth” capture rule.

### Force needs an immutable disposition history and a current projection

Keeping input identity independent of materialized bytes is correct. Define outcome successor semantics before exposing unresolved counts: supported Force after an exact reviewed repair must append an outcome linked to the previous unresolved result, keep the original input bytes, and retire that prior result only from the current unresolved projection. Do not delete the historical outcome and do not count both as current unresolved/supported births.

Identical Force must reuse byte-identical input/outcome records where the disposition is unchanged, with zero duplicate episode votes. Changed extraction under the same episode/ordinal retains both exact input revisions and identifies which disposition belongs to which revision. Specify how branches of an outcome history are selected or refused; digest uniqueness alone does not decide the current state. Non-Force idempotent Apply currently returns zero Stats whenever ep: exists, so durable partial status must remain available independently and must not become “fully ingested” on retry. None of these replay requirements is tested by the current V3 document or the two characterization tests.

## Composition conditions that remain implementation work

The selected B mechanism is compatible with the en:/al:/att: journal allowlist because new births must route every attestation proposal into the transaction-local B buffer. Repeated declarations and later concept/type updates of the same provisional entity must remain in that session; an intermediate en: row cannot switch them into legacy attestation. Existing recognized generations load their own durable evidence; legacy-adopted identities alone retain old att: authority. Baseline/current/history raw references preserve facts but never establish an absent identity's ownership. B and raw-reference primitive verdicts remain the responsibility of their separate graders.

“Capture on behalf of a birth” needs a concrete centralized mutation ledger. It must cover entity replacement, added name/alias keys, alias deletion during revalidation/type refinement, and every allowed claim change. Keep per-key writer/birth attribution in addition to the journal's single transaction-baseline before-image. The journal itself records one before-image, not a history of which birth subsequently wrote the key.

For example, A can provisionally own key k, later drop it, and B can validly claim it. When A is unsupported and B supported, restoring k to the transaction baseline would remove B's surviving claim. V3 explicitly prevents this with independent final-owner validation and atomic refusal where dependencies are unproved; that is a safe policy, not a demonstrated code path or completed implementation. Construct the entire cross-birth final ownership/undo plan before restoring any key. Revalidate every retained listed spelling against the final owner graph after removing unsupported births, rather than treating “key not undone” as evidence. Refusal is safe but remains an ingestion defect as V3 acknowledges.

The baseline full raw scan/count inventory and final full scan must be strict and in one owner transaction, but add exact mutation attribution for all fact writes/deletes/invalidations. The digest is preservation evidence; it is not a support certificate or provenance. A source/target current or historical fact created through the admitted resolver flow may support its newly registered birth. Preexisting references forbid that automatic birth. Unknown/malformed raw rows must still refuse. Serial lead ownership and Badger conflicts are necessary scope assumptions, not a claim that arbitrary raw concurrent writers are prevented.

The exact reviewed legacy inventory is a defensible adoption direction. The stable name/slug/creation anchor permits ordinary description/repo-ref/LastSeen changes without perpetual full-record equality. Mutable type still needs the existing compatibility rules. Every recognized identity use must validate the selector, including protected early returns, all primary and hint endpoints, standalone ordinary mutators and each raw repair transaction. Reviewed merge/retirement must atomically consume losing anchors, account for historical endpoints, and require an explicit lifecycle decision for a reused slug. Missing selectors after adoption are errors; adoption must never grant owners to dangling slugs or silently repair old alias defects. Exact inventory completeness, consumed-state validation, unknown raw fields and the old-writer interval remain separately implementable and untested here. Restored bytes alone do not certify an unaware writer.

The owner-controlled finalizer and poisoning rules address the earlier nested escape if used as specified. No exported caller-selected finalizer, cleanup list, support boolean or legacy-anchor forgery is permissible. The controller must validate final facts, record observations/outcomes, preflight the complete undo plan, restore exact unsupported identity bytes, stage only supported B generations and check all postconditions before commit. Root-captured external writes are not automatically in-scope assertions. Keep all failure/panic cache disposal and zero-on-error Stats requirements.

## Smallest next unit and its bounded bar

Proceed with the private, uncalled outcome codec, immutable writer and bounded readers, using the subject distinction above. It should accept already-captured observations and descriptive resolution/disposition data; it must not select owners, certify support, classify dependencies, write entities/aliases/facts, adopt legacy identities or alter Apply. This isolates preservation from the unresolved controller mechanics. The anchor lifecycle can follow as a separate reviewed unit.

Required independent codec cases: dangling primary and hint-only assertions with no birth; a fact touching two births; exact linked observation existence/key/body/episode validation; wrong episode and wrong role/ordinal links; full raw materialization bytes and absent materialization; nil/empty distinctions and caller buffer mutation; malformed UTF-8/JSON, duplicate/unknown fields and key collision; identical replay and changed revisions; predecessor linkage with no invented ownership; oversized pagination/detail with no skipping; rollback/error/panic/staging failures; backup/reopen exact bytes; no graph events, routing or normal recall effects. Full dependency closure and useful mixed-episode Force belong to later controller integration tests, not codec claims.

No adoption, prevention deployment, cleanup, final performance, recall, real-sweep or whole-goal clause is certified by this review.

## Executed evidence and pins

Executed in the fresh export:

`go test ./internal/memory/resolve -run TestControllerV3 -count=1 -v`

PASS: two synthetic baseline characterization tests. No full suite was needed because production sources were unchanged; no full-controller test was available.

SHA-256:

| Input/source | SHA-256 |
| --- | --- |
| `/tmp/scry-controller-v3-sep06.yKRgWZ/CONTROLLER_V3.md` | `c43e0634e1ee14e8177ab505680e75725c7602abfaefcb7b84a2203c7b60d36c` |
| Archived V2 review, `docs/memory-repairs/admission-controller-v2-independent-review-2026-09-06.md` | `67218c1c362ab20f8a0d328f3a1895935b132c76c33c1e31c5a8f955a3fde49b` |
| `internal/memory/resolve/controller_v3_design_characterization_test.go` | `0ddbb2d4e6c17e2823662f52ce29fe5d2fcd456f5a67fa42e9d88b2b22b49cc5` |
| `internal/memory/resolve/resolve.go` | `43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623` |
| `internal/memory/store/store.go` | `9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891` |
| `internal/memory/store/identity_observation.go` | `83a39c03435185963a4a2931f94f30f03a7bd1a86c6e4c545ece95580ad9e0ab` |
| `internal/memory/store/identity_journal.go` | `d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80` |

Relative source paths refer to `/tmp/scry-controller-v3-review-UMV0Ro`. This report's final hash is supplied separately to avoid a self-hash mismatch.
