# Current-assertion preservation — private experiment, not a drop-in bridge

2026-09-06. Baseline21a7821 (documentation-only successor of e097fa6). No shared
source changed, no original test changed, no real store opened, no providers,
recovery or deployment. Frozen experiment manifest:
`c67a9f82b34b4df20364df7b0051a8258738c3bdd1e7c939d07d9abcccb537b0`.

This experiment pursues the smallest bridge recommended in the independently
reviewed assertion-preservation design, not the optional sibling-address migration.
That design/review is archived in the baseline docs/memory-repairs/assertion-
identity-*. The schema refusal and historical occupied-address parts are already
deployed; canonical current-triple and earlier-start coalescing still discard
different incoming assertions. The read-only identity overlay does not fix them.

## Reproduced baseline defects

New root test03d60f30, five top-level groups, runs through normal Apply:

- Different sentence or raw relation at the same current legacy address silently
  coalesces, marks input ingested and emits events instead of atomic conflict.
- Different earlier/later start coalesces rather than retaining two exact starts.
- Same-episode canonical exact duplicate lowers confidence .95 to .2 and counts
  two additions while only one row remains.
- Exact current restatement with equal timezone-adjusted instant already passes.
- Invalid explicit date can supply an inferred current-restatement address.

Baseline-new-tests.log SHA03785b96 records four failing groups and one passing
group, package.692s. The distinct-start test stops at its first failed offset;
do not claim both offsets were independently observed before correction.

## Private implementation and narrow passing evidence

Three modified sources, all pinned in the manifest:

- Store historical.go56154318: retain HistoricalRestatement's historical-only
  public behavior, add a common exact-address reader for current/historical full
  assertion matches. Canonical raw round-trip required before typed metadata
  rewrite; distinct content at the same legacy address remains ErrFactConflict.
  Reject zero/unrepresentable incoming starts. No new key format or migration.
- Resolver fallback.go2fa29873: all relation restatements require exact address
  and full assertion content/start, not current triple or fallback text-only
  matches. Invalid explicit dates refuse here rather than supplying identity.
- Resolver resolve.gob026d107: recheck same-episode exact repeats for all relations,
  not just fallback. Metadata merging keeps original start/validity, unions EPs
  and takes max confidence. A different start refuses that merge helper; there
  is no DeleteFact/backdate relocation branch.

New tests plus all existing HistoricalAddress tests PASS1.806s in
first-correction-tests.log SHAa6f966ea. Both earlier/later offsets execute after
the correction. This is narrow root evidence, not independent approval.

## Original compatibility failures retained, not fixed by expectation edits

Full original resolver suite FAIL6.357s. Eight groups/ten leaves fail:
StatusIsAlwaysAnAttribute; FallbackPreservesDifferentStatements(two offsets);
MergeRule; MergeRule_EarlierValidFromRelocates; Supersedes_OnMergedFact;
ExclusiveFlip_OrderIndependent(two orders); ValidFromParseFallback/garbage;
RelationNormalized. Log79d60548 retains all output. No original tests were changed.

Some expectations explicitly require discarding a changed sentence/start and are
incompatible with the goal's preservation rule. That does not mean all changes
are justified: globally rejecting malformed dates broadens the former fallback
policy, and the two-phase exclusive/supersedes rules need a complete temporal
and ambiguity contract once multiple full assertions can share a triple.
Do not just rewrite these tests to get a green suite.

The baseline exclusive-order test assumes an undated, different-sentence old-target
mention should merge into the earlier assertion before a same-episode target flip.
Under exact preservation these are separate assertions at the episode instant;
the old Phase B eligibility comparison may leave two contradictory current targets.
This is a source-derived risk to reproduce independently, not yet a root-tested
counterexample beyond the recorded stats failures. Full-assertion identity alone
does not determine which sentence's interval is meant. A new fact planner must
either prove exact temporal effects or retain the ambiguous dependency component;
no text/word heuristic, timestamp nudge or silent collapse is permitted.

Next: use these retained failures to define and independently disprove the complete
FA Phase A/B contract (exact assertion references, original observation links,
historical validity/max-confidence/EP union, full-literal exclusive targets,
eligibility-before-unique SupRef selection, conflicting same-episode inputs and
component-level deferral before ANY mutation). Preserve ordinary date fallback
where it does not falsely identify an old assertion, unless separately justified.
An additive exact FactRef/sibling format still requires an all-consumer transition,
old-writer floor and actual backup/rollback proof before recovery. This experiment
implements none of those and must not be integrated or deployed as-is.
