# Private complete episode selector — implementation contract

2026-09-06. Baseline e13f7ae; no birth-registration implementation dependency.
This contract incorporates the COMPLETE frozen design3fd0d796 and COMPLETE
independent design review3ee87a6b, with the corrections below taking precedence
over contradictory or underspecified draft text. Both referenced files must be
read in full. No production caller, semantic support/ownership, real Force/status,
adoption or live deployment is included.

Implement private io-episode:/io-head: V1 records, exact input/observation/outcome
cross-validation, finalizing-only immutable writer and head CAS, descriptive counts,
owned selected-result reader/token and complete bounded immutable result chunks.
Add both families to the adopter's reserved-prefix refusal only. No other existing
source or test contract changes; old outcome mixed-revision/permutation/predecessor
characterizations remain unchanged and are not relabeled selector successes.

## Exact API and coverage

Writer takes a genuine live finalizing admission facade, full EpisodeID, exact
expected head state (existence/raw), and a deeply owned complete proposal. Proposal
contains InputKey; ordered declaration/assertion/birth entry values as designed;
and a nonnil list of complete proposed outcome values. All proposal durable
OutcomeKey and result Predecessor fields must be empty: writer derives them.
No body/finalizer callback, support boolean, caller baseline or commit certificate.

Associate each proposed outcome with exactly one listed complete birth tuple or
committed/deferred assertion ordinal. Each listed birth requires one outcome;
each committed/deferred assertion requires one; no other assertion permits one.
Reject duplicate, missing and EXTRA outcomes, even if an unused one validates.
Birth tuple lookup may use a derived ID for indexing but full tuple bytes must
match. First-observation identity remains separate and exact. All counted entries
have ordinal equal to their actual slice position, not merely any ordinal present
somewhere in the revision. Validate every persisted observation's key/raw, exact
slot origin/ordinal/side and real EP provenance against full pinned input bytes.

Declarations are identity-mention/value-mention/deferred-identity; value/deferred
birth refs empty. Birth refs must exist and match actual declaration-role links
bidirectionally. Assertions are committed(empty reason)/deferred(identity-dependency
or assertion-dependency)/non-assertion(empty-relation or two-values)/unresolved
(missing-source or missing-destination). Both primary observations mandatory for
every assertion. Committed/deferred outcomes require both primary roles, correct
ordinal/disposition and, if Supersedes is nonnil, both supersedes roles. No
non-assertion/unresolved outcome masquerades as a committed V1 outcome.

Births are duplicate-free by slug, full tuple and first observation, with exact
matching tuple/first primary or declaration link. Order by first observation:
declarations then endpoints, ordinal then original side src before dst. Match
ALL additional birth links to pinned input. This is structural inventory only;
actual registration completeness, retained previous Force inventory, accepted
metadata and all disposition semantics remain future fixed-finalizer obligations.

## Canonical ordering before semantic comparison

Normalize OWN proposed outcome Links: decode/match every observation first, sort
by origin (declaration before endpoint), ordinal, side (src before dst), then role
declaration/primary-src/primary-dst/supersedes-src/supersedes-dst. Duplicate Key/Role
refuses; retain every annotation/materialization byte. Canonical encode/decode
normalizes UTC birth values. Sort declaration birth-reference lists by canonical
Births order after duplicate/reference checks. Do not reorder declarations,
assertions or Births silently; their required canonical ordinal/order must hold.

Selected persisted outcomes must ALREADY have canonical link order; validate and
refuse noncanonical rows instead of rewriting/adopting older V1 outcomes. New
selector normalization does not alter the old outcome encoder/lineage contract.

## Semantic reuse, historical deduplication, head checks

Incoming outcome predecessors must be empty. After normalization, compare complete
canonical semantic bytes (predecessor cleared) to at most one CURRENT selected
same-lineage outcome (full existing outcomeLineage bytes). Equal reuses current
key. Changed same lineage uses that exact current key as predecessor. Absent or
different lineage starts empty. No arbitrary history search or predecessor choice.

An internally derived content key may already contain exactly the same historical
bytes, especially A->B->A when B omitted A's lineage. Permit full-byte-equal
immutable deduplication at that key. This is not selecting historical authority.
Episode result still points to immediate B and head increments. Do not require
a novel outcome key or add a nonce merely because content appeared before.

Only after resolving outcome keys compare full canonical episode semantics with
Predecessor cleared. Unchanged current semantic result yields zero writes and the
same exact head/result. A changed result gets immediate current ResultKey as
predecessor and exactly one checked revision increment. Check expected existence
and raw bytes BEFORE every no-op decision; stale identical requests refuse.

Reader proves LOCAL head structure only: positive uint64 revision; revision1 iff
selected result predecessor empty; revision>1 requires valid distinct same-episode
immediate predecessor. Validate that predecessor's COMPLETE own record/input/
observations/outcomes against ITS input revision, but do not recursively walk its
episode predecessor chain. Do not claim head counter equals full history length
or detects arbitrary raw-store tampering. Writer proves exact current CAS and
checked +1 for this transition; overflow refuses. First selection is revision1.
Legacy ep without head is unselected, not known zero-unresolved or fully ingested.

## Transaction/report and inspection boundary

Check genuine phase/facade/transaction and prior poison before cached results.
Sanitize local writer validation/storage errors to a static sentinel plus only
TxnTooBig/Conflict, poisoning even if caught. Deep-copy inputs before retaining
them, preflight complete proposals and old state before staging, compare occupied
immutable rows with full bytes. Stage outcomes/result/head atomically in the
existing outer transaction. Writer returns OWNED STAGED descriptive data only,
never Applied/Committed success; a later outer conflict can invalidate it. Future
fixed outer wrapper must sanitize actual commit error and return no committed
report on error/panic. Observer panic is postcommit under existing semantics.

Read selected state in one view, owned copies, derived counts only. Fact counts
sum to input facts; birth counts sum to described inventory. No user-visible
fully-ingested flag or current live-identity ownership claim. Selection token has
head key, raw-head digest, positive revision and fixed-size result key, not a full
potentially long EpisodeID. Token is inspection only, NEVER a raw-head CAS token.
Chunk reads pin exact immutable result key/episode and validate complete record;
full internal JSON/base64 envelope must fit caller budget512..24576 with forward
progress/EOF and no silent skip. Current head may change during reassembly without
changing the pinned bytes. No full public RPC/CLI 24KB claim before integration.

No graph/fact/identity writes, votes, alias authority/undo, support/dependency
policy, producer routing, queue/orient/Force integration or live adoption. Complete
design/review test matrices apply. Root is builder only; independent code disproof
and full no-CGO suite required before retaining this source.
