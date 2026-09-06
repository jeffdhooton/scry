# Actual owned materialization — private progress

2026-09-06. Supersedes the earlier conflict-only implementation status, not its
evidence. Root private `/tmp/scry-complete-admission-sep06.6qYb7K/code` now connects
normal first-input, same-input and proven-none-accepted replacement to the SAME
fixed `applyCompleteAdmission` transaction. Main production and live binaries are
unchanged. No actual memory replica or live store was used for this stage.

The shared internal program performs the same owned discovery, complete assertion
planning, dependency/support/ownership closure and receipt construction as the
read-only wrapper. The actual writer stages approved final EN/AL through identity
accounting and FA through fact accounting, preserving exact raw before-images. It
persists original EP/input/observations, supported lifecycle and actual vote-domain
rows, immutable complete result and selected head. Events remain buffered until
commit. Caller provides no plan, support, mutation list or finalizer.

The intermediate ledger freeze still covers its original EN/AL/FA domains. A fixed
finalizer separately validates genuine retained incoming-FA birth support and exact
generation records, then stages lifecycle/votes and result/head under complete raw
delta accounting. Full actual rows and events are compared before and after these
final writes. This is explicit additional coverage, not a claim that the earlier
identity ledger already authorized IG writes. Same head is reused on an unchanged
result; changed selection checks exact old head and predecessor. Late result-row
refusal rolls back previously staged EN/FA/IG/EP/input and emits no success/events.

Adjacency receipts now retain exact key/presence/full before and approved after.
Existing FA rewrites preserve stale/missing/opaque mirrors rather than silently
repairing them. New FA can create an absent nil mirror or preserve an existing nil
one; an opaque conflicting mirror defers local effects. Replay verifies mirrors
actually owned by a prior new-FA effect, while an old metadata-only FA receipt
does not forbid an independently repaired old mirror.

## Retained failures and corrections

All source/test first failures remain in private6qYb7K; no supplied independent
tests were edited. These are root implementation tests, not independent approval.

- First adjacency cases failed .595s, log
  3155bcec89984625dc586c4fc517ce80c0203202e8ffe4ca8511d10fcf11df53.
  Source-only fix selected suite passed 6.796s, log
  814b846410febbda9a5f031fad7be72c49c02ba66c0031af614f41c69e692e18.
  Actual terminal mirror disappearance then failed .843s, log
  37b50b12c33c08b0ea43809209bc0709751865252087b7957b48006044208aa9;
  previous-reader correction passed 5.097s, log
  f565d8bca83ac202286655897447f254102ad47d78591acddac5b064f7185da1.
  The old-missing-mirror fixture explicitly restores original absence after its
  generic seed writer, which otherwise repairs it. No production policy change.
- Initial normal-entry tests failed against the temporary refusal, log
  4c4ab39c851bf8e6710a4a99287d9d77ae3115f05c29874f82f6b6e7b9f2e1d0.
  First actual implementation committed new rows but refused exact zero-event
  replay because nil and empty event slices compared unequal: .933s, log
  2d51645ac5f6ec2a526ed2ebdd9edf3cd01bd80204cb141abe8acfbbbf5a6659.
  Source-only elementwise full event comparison passed 1.343s, log
  965a20f66e6874b18c09f4eb293c3c001a8aebe57e14fea7df29bc757ac236d0.
- Actual partial own-commit retry unnecessarily selected a new head because fresh
  deferred evidence saw the prior accepted writes: .539s, log
  ea61c412cfe622d0ef41f60bc16cfea1ec64821068f556330b8d49b707404c17.
  Replay now retains an unchanged deferred assertion's exact old receipt when
  full semantic content matches, excluding only fresh evidence/trace/producer and
  predecessor bookkeeping. Original observations, routes, full FA/address images,
  hint matching/eligible sets, targets, intervals and components remain compared.
  Fresh planning evidence remains owned; old receipt is not execution authority.
  Unchanged test plus selected replay suite passed 1.822s, log
  6ad7cd162165bef98b93e5d5d8b8aa151f3fb82f2ebc89056682ece6364966d2.
- A real selected revision with proven zero acceptance could not be replaced:
  .616s, log8b18242b75db5f53374bbfc0d4964d89eac243bd1bf35a5d7b01af31349f4773.
  New-input planning now binds actual old head/lineage but carries no old occurrence
  or action program when no acceptance is established. Unknown or any accepted
  effects still prohibit replacement. Source-only correction passed 1.736s, log
  94046a70655629afb68300bdbd86c732ba17f8366401fb9d23e610324aae328d.

Additional actual entry tests verify late selection rollback, duplicate vote
deduplication in legacy and generation domains, no unsupported birth/vote, retained
terminal effects and later pending-action progress without metadata refill.
Normal/conflict tests passed 1.387s, log
9ce0b21a9c0240af5d02e36cc060131e1638491b2c486396e7979388777d94b6.

Full uncached no-CGO suite passed store69.325s/resolve17.829s/daemon28.979s,
logdbf3b3db1e9471cfe2bfee0986b8e906398449d7908fc282488426e1a3389c15.
Relevant vet passed. Explicit private TMPDIR used.

## Independent review freeze and ongoing work

Exact frozen export `/tmp/scry-owned-commit-review-sep06.xfHydo/code` contains
600 unchanged e097fa6 baseline files plus 99 additions. All source/destination
hashes verified. OWNED_COMMIT_FREEZE.json SHA-256
a8976cf6bb9c99b0e9d76034781647e74048f458903461034540275c70f4bbec;
complete controlling contract20d5ed6f copied exactly. The frozen source includes
four unchanged earlier independent tests. Fresh-context owned_commit_disproof is
testing actual transactions in a separate private export. No verdict yet.

After freezing, root added only a new backup/restore/reopen test to its working
export, leaving the review snapshot immutable. Supported generations and votes,
legacy votes plus partial results, and unsupported declarations preserve every raw
row, complete selected result, exact head and zero replay events after real reopen.
PASS .747s; test493c0c707d379ab8e39bdd0ec96229922cd00e4df988839a5750bcbb95e1332e;
log87760bbad3d8e397edf655f157d7e52685fe930cbdc49b750fcca85936cdedc1.
These are synthetic backups, not an actual shared-memory snapshot certificate.

Normal resolve.Apply adapter/policy deduplication, public V2 status, all-writer and
schema-floor enforcement, actual-backup conservation/adoption, complete independent
review and every original live acceptance bar remain. Private normal writer is
real implementation progress, not delivered live prevention. No new deploy, live
write, actual backup/probe, durable note or full-goal grading round occurred.
