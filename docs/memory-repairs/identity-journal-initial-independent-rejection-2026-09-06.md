# Frozen identity journal primitive: FAIL

2026-09-06. Independent private primitive review, not admission/prevention approval
or a whole-goal verdict. Session orientation, the complete active goal, initial
independent admission design, and complete delayed-evidence correction were read.
No production files, live store, providers, configuration, queues, binaries or
deployment were changed. Tests use disposable synthetic stores only.

Base: independent `git archive a06cd7b` export, full commit
`a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897`, at
`/tmp/scry-identity-journal-grade.IQZmSR`.

Exact frozen candidate SHA-256:

- `internal/memory/store/identity_journal.go`:
  `929fcc464e961e838cf9245b14d0cde9e4ac287a4c85e28bf1f51a29391e84d9`
- Builder's unchanged `identity_journal_test.go`:
  `c4eda2110d6a45b025b4119e279292aceb760c4d2722103b0d13656aea404ab8`
- Independent `identity_journal_disproof_test.go`:
  `a3ac5d16a528bb87a5c182814abe57024c0bbbb103f7270fd70be97b081b29bd`
- Independent `identity_journal_boundaries_test.go`:
  `89228ee78c1af3b4144b04384d073ce06885e7999b5153bc21779c6d391ad0aa`

## Blocking counterexample

`restore` at candidate lines 121 and 123 passes caller-owned `identityUndo.Key`
buffers directly to Badger Set/Delete. Badger retains those buffers until the
transaction completes (installed Badger v4.9.1 txn.go:398-430). The primitive
neither owns those bytes nor documents an extended input lifetime requirement.

The two independent tests capture `en:opaque`, stage a change, successfully
restore that exact whitelisted key, then reuse the undo key buffer by copying
`fa:opaque` into it after restore returns but before the outer transaction returns.
All primitive operations return success. No races, ignored errors, journal-internal
mutation, or changing keys during preflight are involved.

- With an existing entity before-image, the unrelated synthetic `fa:opaque`
  record is overwritten by identity bytes.
- With an absent entity before-image, the unrelated synthetic `fa:opaque`
  record is deleted.

These are intentionally raw key-family tests, not a claim that the fixture uses
a semantically valid serialized Fact. They disprove the primitive's whitelist and
exact-key boundary without needing admission semantics. The caller only gives
the primitive an entity key. Direct Badger fixture writes retain their own buffers.

Reproduce from the private directory:

```sh
go test ./internal/memory/store -run TestDisproofJournalRestoreRetainsCallerKey -count=1 -v
```

Both tests FAIL with the expected messages identifying the fact overwrite and
deletion. This frozen candidate remains unmodified so the counterexample stays
reproducible. The lead was informed before changing or grading any new candidate.

Smallest fix: copy undo keys into an owned immutable preflight/staging plan, and
pass only plan-owned buffers to Badger Set/Delete. Owning expected and before-image
bytes too makes the plan lifetime explicit. Do not merely clone the whitelist
lookup string while still passing the original slice to Badger.

## Other results and limitations

Before adding independent tests, `go test ./...` passed the full unchanged a06cd7b
repository suite plus the three exact builder tests. After independent additions:

```sh
go test ./internal/memory/store -run 'TestGradeJournal|TestIdentityJournal' -count=1 -v
go test ./internal/memory/resolve -run 'TestApplyReleasesTransactionalCompactCaches|TestApply_EmptySlugSkipped' -count=1 -v
```

Both commands PASS: five independent boundary tests, three builder tests, and the
two unchanged resolver regressions. The independent tests establish capture-key
copying and ValueCopy isolation, first-capture idempotence, exact preservation of
opaque JSON including duplicate unknown fields and whitespace, absent versus
zero-byte values, duplicate/forgotten/disallowed-key/stale-value precondition
rejection with no partial staging, complete event-suppression preflight,
preservation of existing/unlisted entity events and fact/episode events, and
suppression only of explicitly named captured-absent entities still absent now.

An oversized captured key deliberately makes the second Badger Delete fail after
the first undo entry has staged. Returning the error rolls back entity, alias and
events. This distinction is material: precondition errors stage nothing; storage
errors may stage a prefix but require the caller to abort the outer AtomicWrite.
Silently ignoring errors or committing after a journal error is outside the
documented contract and was not counted as an implementation defect.

Nested AtomicWrite reuses the same facade. A panic rolls back writes and emits no
callbacks. Every escaped journal method rejects use after panic or successful
commit with ErrDiscardedTxn, including while a later transaction is active.
The existing compact-cache lifetime regression covers unchanged resolver success
and error paths. No journal/resolver integration exists, so there is no new combined
admission/cache execution path to certify.

Capture records the first image at capture time, not an independently recovered
transaction-start image. Capturing after the corresponding mutation, omitting a
touched claim, mutating after finalization, or treating event suppression as proof
of factual support is a caller-contract violation, not protection this primitive
provides. Journal restore does not automatically plan or suppress events.

Even after the buffer bug is fixed, this primitive cannot be integrated as hollow
prevention or repair. Generation-bound persistent admission evidence and a durable
selection boundary are still missing. Transaction-local isolation cannot prevent
old orphan attestations regaining authority after a supported identity commits;
legacy saturated lists also cannot retain new independent evidence. Structured
observation metadata, all-current/historical-fact support validation, caller
coverage, and the legacy EmptySlugSkipped contract reconciliation remain open.

No live-scale cost, snapshot restoration, external-name corpus, recall, live
hollows, deployment compatibility, or whole-goal clause is graded by this report.
