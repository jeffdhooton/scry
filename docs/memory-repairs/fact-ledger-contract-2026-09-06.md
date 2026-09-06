# Private exact fact mutation ledger proposal

2026-09-06. Implementation/review next; not a deployed policy. Combine with the
reviewed owner boundary and complete raw reference inventory, without changing
their contracts. This unit must not select support, certify parsed assertion
semantics, update identity/alias/evidence rows, or authorize maintenance.

Begin only on a live admission-owned facade in body phase, before production
fact mutations. It obtains its own strict complete baseline reference inventory;
the caller cannot supply a baseline digest or a support boolean. Capture the exact
facade/owner identity; never transfer/reuse the ledger across owners/phases. Any
ledger failure poisons the owner, including ignored errors and storage failure.

Private ledger put/delete are the actual fa: transaction writers. For each call,
validate the raw key/body with the strict checker, read exact current bytes from
the active transaction, and retain owned before/after bytes for touched keys only.
If that key was already tracked, its actual current state must equal the last
tracked after-state before another mutation. No bare key prefix or decoded Fact
can substitute for complete canonical endpoint/time validation. Deletion requires
an existing valid fact. Record every successful mutation's original occurrence
ordinal (nonnegative) in global execution order; a repeated ordinal is legitimate
for merge/relocation/invalidation. The ordinal is descriptive attribution, not
proof that a resolver supplied the correct original occurrence.

Only fa: is written by this unit. Adjacent-index writes and observer events remain
the responsibility of existing normal methods when these helpers are integrated.
No such integration exists in the first unit, and no production caller is added.
Do not treat raw ledger put as permission to replace a distinct existing assertion
or drop facts. Normal preservation guards and exact relocation semantics must wrap
it in the eventual all-writer policy; the low-level nudge path is not approved.

Final verification runs once, in the same owner's finalizing phase, and revalidates
every complete final raw fa: record. Verify that the final bytes/presence of every
touched key equal its last tracked mutation. Reconstruct the baseline digest in
exact ordered framing: substitute each touched key's first before-image, reinsert
deleted original keys, omit new keys, and leave every untouched final key as-is.
Both reconstructed count and digest must equal the captured baseline. This detects
any untracked final insertion/removal/byte change, including a change that happened
before a key's first tracked write. Do not claim detection of transient raw changes
that were reverted byte-for-byte, arbitrary concurrent snapshot phantoms, or raw
changes performed after verification by a private finalizer.

Return an owned final count/digest inventory and owned ordered mutation records
with exact key/before/after and ordinal; retain no complete raw baseline map or
Fact slice. The future Store-owned fixed finalizer must check actual allowed
occurrence/dependency semantics, select baseline-zero births, and invoke terminal
verification after its last possible fact mutation. This ledger is write-set
accounting, not a support certificate. No public caller-selected finalizer/list.

Error messages are static and never wrap arbitrary decoder/storage errors; only
static ErrTxnTooBig and ErrConflict classifications may survive. No partial reports.
If the owner was already poisoned elsewhere, the ledger returns its own static
refusal without exposing that earlier error. The outer owner still retains its
original first failure under its existing contract; this unit does not claim to
sanitize errors from unrelated ordinary mutators or change their error precedence.
Immutable caller buffers, nil/empty/absent distinctions, exact rollback, no graph
events from these private helpers, successful commit and all caught-error paths
must be tested. Real size failure after prior successful writes must roll back.

Required tests: historical/current/attribute/self-loop writes; unchanged/noop and
repeated ordinals; known-key put/delete/reinsert; original deleted keys interleaved
before/between/after retained keys; untracked modifications before first/after last
record and on untouched keys; new/old raw unknown extensions; malformed baseline
and final records; forged/wrong owner/phase/closed/repeated verification; caller
buffer/report mutation; cross-transaction isolation; exact error/event/raw proof.

Source-map obligations before eventual production integration:
PutFact fa:Set; InvalidateFact fa:Set; DeleteFact fa:Delete; RelocateFact oldDelete
and newSet; resolver mergeFact's DeleteFact+PutFact; historical-address preservation;
PhaseA merge and PhaseB Supersedes/exclusive invalidation. Reviewed raw merge and
retire writers use their own validated lifecycle operation and remain refused in
an admission-owned facade. Identity/alias journal attribution is a different ledger.
PutMetaTime/PutMetaJSON must protect reserved lifecycle marker names separately.

Baseline0995859. Root implemented this contract privately as
internal/memory/store/identity_fact_ledger.go; production integration remains absent.
The verification implementation performs the reviewed complete final scan followed
by a same-transaction ordered reconstruction pass. It does not claim single-pass
performance or reuse a mutable cached inventory. Private struct-field corruption
by arbitrary same-package code is not a security boundary; owner/facade/phase and
zero/nil/foreign-capability controls are the executable reuse boundary tested here.
