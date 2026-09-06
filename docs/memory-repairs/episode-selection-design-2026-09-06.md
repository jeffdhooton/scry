# Complete episode selection — concrete private contract proposal

2026-09-06. Baseline e13f7ae. Design only; no selection implementation, production
caller, successful-ingestion certificate, Force integration, lifecycle adoption,
or live change. Refines current design39b32ab7 and its complete reviewa9722fcf.
Input revision b69e6daf is now available. Birth registration509faba4 is separately
under code review and is not assumed approved here.

## Why this unit

One complete episode result selects its entire declaration, assertion and birth
projection. Immutable per-occurrence rows alone leave omitted old occurrences
falsely current after changed extraction. The selector must preserve all old rows
and distinguish unresolved work from graph ingestion, while making exact retries
write nothing. This unit validates structural records and exact selection only;
the fixed admission finalizer will still have to prove every proposed disposition.

## Frozen record shapes for review

Propose two new families, `io-episode:` and `io-head:`. Keys are respectively
prefix + SHA256(episode ID) + ':' + SHA256(canonical complete result bytes), and
prefix + SHA256(episode ID). Hash matches never substitute for full byte checks.
Keep schema1; add both exact families to legacy adoption's reserved-prefix refusal
with malformed/present-empty coverage. All-writer lifecycle protection is still
a required integration gate, not provided by this codec.

Episode result V1 has Version, full EpisodeID, InputKey, ordered Declarations,
ordered Assertions, ordered Births, and Predecessor (a result key or empty).
Top-level classification arrays must be nonnil, including empty arrays. Their
lengths derive from the pinned input; never collapse nil/empty input arrays, which
are retained exactly in the separate revision. Result arrays describe zero entries
for either nil or empty input, but InputKey keeps those revisions distinct.

Each declaration entry has its zero-based Ordinal, exact ObservationKey, and Kind:
`identity-mention`, `value-mention`, or `deferred-identity`. It includes a nonnil
ordered list of birth FIRST-observation keys to which this declaration is linked;
zero is allowed (existing identity/value/deferred without a new birth). Duplicates
refuse. Links to births must also occur as declaration links in those births'
outcomes. Value/deferred entries require zero birth references. Do not invent a
birth from the declaration's natural slug or treat identity-mention as proof that
metadata/alias proposals were approved or completed. This records every original
declaration, including repeats and conflicting metadata, without claiming policy.

Each assertion entry has its zero-based Ordinal, Kind, Reason, exact primary source
and destination ObservationKeys, and OutcomeKey. Kinds/reasons are closed:

- `committed`: empty Reason and matching V1 committed assertion outcome.
- `deferred`: Reason `identity-dependency` or `assertion-dependency`, matching V1
  deferred assertion outcome. A deferred supersession dependency needs no birth.
- `non-assertion`: Reason `empty-relation` or `two-values`, empty OutcomeKey.
- `unresolved`: Reason `missing-source` or `missing-destination`, empty OutcomeKey.

Every input fact ordinal appears exactly once in original order, including the
empty-relation skip and endpoint failures occurring after another stub was staged.
Both primary observations are mandatory even when an endpoint string is empty.
For committed/deferred outcomes, links must include those exact two primary roles;
when original Supersedes is nonnil, both supersedes roles must also appear, carried
by one or both exact matching primary observations. All linked inputs are checked,
not just those needed to find one birth or primary source. Non-assertion/unresolved
retain the complete original Supersedes through their exact primary observations;
absence of an outcome is never treated as a committed assertion.

An unknown relation may normalize to a valid fallback; it is NOT automatically
empty-relation. This codec only validates the closed description, not the semantic
truth of any classification/reason. The eventual fixed policy must produce and
verify these choices against actual resolution, dependency and mutation evidence.

Birth entries carry the complete Birth tuple, FirstObservationKey, and OutcomeKey.
The list is ordered by first occurrence traversal: declarations by ordinal, then
facts by original ordinal and src before dst. This is an explicit canonical result
order, not inferred mutation order. Reject duplicate slugs, first keys, or birth
identities. Every birth outcome must have exactly the same full tuple, have a
primary/declaration link to the exact first key with original role, and have every
additional observation link match this result's pinned input revision. Cross-check
declaration backreferences in both directions; endpoint links need not originate
a birth, but all endpoint observations are already covered by total assertions.

This exact duplicate-free birth list is a STRUCTURAL registration inventory, not
proof that the caller captured all creations. During admission the Store-owned
finalizer must compare it against genuine registration and selected prior-result
evidence. Current transaction registrations alone are explicitly insufficient.

## Replay birth coverage without inventing births

Do not store `fresh` versus `retained` in the semantic result. On identical Force,
a formerly supported birth is now an existing recognized entity; that does not
remove it from the selected result. The eventual finalizer must carry forward an
exact prior selected birth/outcome whose first and all linked observations still
match the pinned revision, after separately validating current lifecycle state.
A previous no-assertion birth can likewise remain a descriptive prior occurrence;
it is not a live selector or permission to reuse its slug. Any revised outcome must
be based on actual new admission evidence, not name/frequency matching.

The private selector accepts the complete proposed inventory and rejects structural
holes/duplicates; it cannot infer completeness from live entity names. Synthetic
replay tests supply the same proven structural inventory on exact retry, and must
explicitly demonstrate that registration-only enumeration would change the result.
Different input may omit old births/ordinals from CURRENT selection, but never
authorizes deleting facts, identities, votes, observations, or old results.

## Semantic outcome reuse before episode equality

The private write request carries proposed full outcome values, not caller-chosen
new predecessor chains. Incoming outcome Predecessor must be empty. It carries one
outcome per referenced birth and committed/deferred assertion, addressed locally
by exact first observation identity or exact assertion ordinal. Construct durable
OutcomeKeys internally after validating all proposed outcomes and their complete
links against the exact persisted InputKey/raw revision and real episode row.

For each proposed outcome, locate at most one outcome of the CURRENT selected
episode result with the same exact existing `outcomeLineage` bytes. Compare full
canonical outcome bytes with Predecessor cleared. If equal, reuse that selected
OutcomeKey without writing a successor. Otherwise set Predecessor to that exact
current same-lineage key and append the changed outcome. If lineage differs or is
absent, start an empty predecessor; never forge a cross-revision occurrence link.
Do not search arbitrary old history for a more convenient predecessor or vote.

Then compare canonical episode semantic bytes with Predecessor cleared: exact
InputKey, all ordered declaration/assertion entries, and full ordered birth entries
including the selected/reused outcome keys. Equal means reuse the current result
and exact head bytes with no writes. Different means append one result whose
Predecessor is exactly the expected selected current result, and advance once.
A -> B -> A retains the intervening transition: no return to the old head revision
or arbitrary historical result. Historical outcome keys may be reused only when
they are the currently selected same-lineage unchanged outcome as above.

All comparisons use full bytes, not digests, map order, timestamps, or counters.
The request is deeply copied and structurally preflighted before staging. Even a
semantic no-op must validate canonical expected/actual head and all linked input,
observation and outcome records first; corrupted old state is not a cache hit.

## Exact head transition, atomicity and missing state

Head V1 contains full EpisodeID, selected ResultKey, Revision uint64 and Version.
Canonical key/body required. A write includes exact expected head existence plus
owned raw bytes. Absent expected requires nil bytes; present expected requires full
canonical bytes. Actual existence/raw must match BEFORE deciding semantic no-op.
Stale expected bytes always refuse, including an otherwise identical result.

First head has Revision1 and result Predecessor empty. A change requires checked
current Revision+1 (overflow refuses) and new result predecessor exactly equal to
current ResultKey. Validate the existing head, linked complete result, exact input,
all observations/outcomes and immediate predecessor in ONE transaction. Reject
missing, malformed, wrong-episode, or noncanonical linked rows. No rewrite/repair
of malformed selectors. An existing ep without a head is `unselected`, not fully
ingested, zero unresolved, or automatically migrated.

Expose only an unexported writer on a genuine live finalizing admission facade;
it is a component for the future fixed finalizer, not a new public caller-selected
success API. No callback/finalizer injection in this unit. Errors sanitize to a
static local sentinel with allowlisted TxnTooBig/Conflict and poison the owner even
when caught. Check phase/prior poison before idempotence. Immutable occupied rows
require complete byte equality. Stage all outcomes, result and head in the same
outer transaction; failures/panics roll back together. Returned staged data is not
a commit certificate; only the outer fixed wrapper may return it after commit.
Postcommit observer panics retain existing coordinator semantics.

Reader validates one complete selected result in a single read view; returns owned
copies and derived counts, no stored counters. Count each original fact once, not
rows, births, invalidations or linked hints. Report separate committed, deferred,
non-assertion, unresolved, declaration-deferred, birth-supported and birth-no-assertion
counts. No status flag equates a structurally valid result with successful graph
ingestion; choosing that user-visible semantics requires the independent utility
grade and real resolver integration.

## Inspection and exclusions

For this next unit, keep readers/writers private. Pin an immutable ResultKey for
subsequent detail inspection; never reread current per page. Add a lossless bounded
byte-chunk reader for validated result records and a small selection token containing
head key/hash/revision and result key, not an unbounded full EpisodeID. Do not claim
24KB RPC compliance from an internal byte budget. Future public command must test
its COMPLETE envelope, long IDs, escape/base64 expansion and head changes across
all pages. Neither records nor dispositions enter alias, graph recall or embeddings.

No graph/identity/fact mutation, generation evidence, support/dependency decision,
alias undo, real Force classification, queue/status/orient integration, admission
activation or adoption is authorized by this unit. All existing counterexample
tests remain unchanged, including mixed old V1 outcomes and predecessor-only churn
accepted under their older finite contracts. New selector must reject mixed pinned
revisions and suppress that churn without weakening those codecs.

## Requested independent design disproof

Challenge executable completeness, bidirectional declaration/birth coverage,
all-observation matching, closed skip versus unresolved counts, exact old lineage
reuse before result equality, identical Force inventory, A/B/A and stale no-op CAS,
head/result predecessor validation without invented history, caller-buffer ownership,
long-ID bounded inspection, and private finalizer-only storage-error poisoning.
Flag ambiguity requiring a smaller frozen contract before code. This is design
review only, not an implementation PASS or full-goal grade.
