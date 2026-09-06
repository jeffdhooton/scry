# Corrected identity journal primitive: bounded PASS

2026-09-06. This is an extension of the independently preserved frozen failure at
`/tmp/scry-identity-journal-grade.IQZmSR/REPORT.md` (SHA-256
`cfda42a189cb78e0d979771c61e43322d8dc86d3886267a7f7179341dd8290c5`).
The old candidate and both failing reproducers remain unchanged there. The lead,
not this grader, made the fix. The new candidate was copied with apply_patch into
a separate fresh `git archive a06cd7b` export at
`/tmp/scry-identity-journal-fixed-grade.xiYvzO`.

## Exact pins

- Base commit: `a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897`.
- Corrected `internal/memory/store/identity_journal.go`:
  `d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80`.
- Unchanged builder `identity_journal_test.go`:
  `c4eda2110d6a45b025b4119e279292aceb760c4d2722103b0d13656aea404ab8`.
- Unchanged independent `identity_journal_disproof_test.go`:
  `a3ac5d16a528bb87a5c182814abe57024c0bbbb103f7270fd70be97b081b29bd`.
- Unchanged independent `identity_journal_boundaries_test.go`:
  `89228ee78c1af3b4144b04384d073ce06885e7999b5153bc21779c6d391ad0aa`.

## Outcome

The attempted disproof fails against this exact corrected private primitive.
Both original tests now PASS without assertion changes: reusing the caller's
undo key buffer after restore returns no longer overwrites or deletes the
unrelated synthetic fact-family key. The corrected method builds an owned plan
before any preflight reads/staging: it clones keys, expected values and old
before-image values, and passes only plan-owned buffers to Badger.

All five other independent boundary tests and all three original builder tests
also PASS. They cover capture-key/value independence; first-capture idempotence;
opaque unknown JSON and exact bytes; absent versus existing empty bytes;
duplicate, forgotten, disallowed and stale expectations; no partial restore on
precondition failures; no partial event filtering on mixed invalid suppression
sets; preservation of existing/unlisted entity and fact/episode events; suppression
only of explicitly named captured-absent entities which remain absent; storage
errors returned to outer AtomicWrite with complete rollback and zero events;
nested facade reuse; panic rollback; and rejected use after commit/panic or in
later transactions.

Commands:

```sh
go test ./internal/memory/store -run 'TestDisproofJournal|TestGradeJournal|TestIdentityJournal' -count=1 -v
go test ./...
```

The targeted command reports ten PASS tests. The full repository command PASSes
all packages, including unchanged resolver, daemon, queue, store, and recall
tests. Complete output is saved in `full-suite.txt`; none of its package results
were marked cached. No preexisting repository test or assertion was modified.
The full suite includes the unchanged compact-cache lifetime and EmptySlugSkipped
baselines. The latter still requires a hollow stub under ordinary Apply, which
confirms this uncalled primitive has not changed the normal admission contract.

## Strictly bounded interpretation

PASS means no further counterexample was found to the exact primitive contract
in these independent tests and source inspection. The whitelist is en:/al:/att:
and before-images are byte-preserving. This is not a semantic entity undo,
metadata evidence system, fact-support classifier, or automatic rollback guard.

The caller must capture before each corresponding mutation, provide the complete
explicit expected-byte undo set, stop identity decisions after finalization,
return every journal error to abort the outer transaction, and independently
prove support/preserve useful metadata/restore claims before suppressing events.
Storage staging errors can leave a transaction-local prefix restored until outer
rollback; this was deliberately exercised and is within that documented contract.
Capturing late or deliberately committing after an error is not made safe.

No production caller references the new journal. There is no numeric schema
change, persistent format, alias generation, structured observation record,
queue behavior change, live write or deployment. Its first builder test removes
new synthetic uncommitted attestation records solely to exercise raw undo; that
does not approve dropping useful admission evidence.

This is NO-GO for presenting journal-only work as hollow prevention or cleanup.
The complete delayed-evidence correction remains required: supported new
identities need a persistent generation selector and independent generation-bound
attestation storage; old orphan counts must stay without authority across later
episodes and saturated legacy arrays must not consume fresh-evidence capacity.
Structured observations, historical/dangling fact support, all production caller
coverage, useful unresolved evidence inspection, legacy-test reconciliation,
retained-writer compatibility and independent integration review remain open.

No live replica was needed for this bounded raw-key failure/fix. No throughput,
live-state audit, recall, old-hollow retirement, status-value prevention, deployment
or whole-goal bar is certified. No live data/provider/configuration/queue/daemon
operation or shared checkout edit occurred. The recommended next step is to retain
these exact regression tests with the primitive before separately reviewing any
persistent admission design; no further primitive fix is proven necessary here.
