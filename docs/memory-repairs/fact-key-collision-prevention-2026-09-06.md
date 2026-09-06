# Canonical occupied-key overwrite: prevention in review

Status: exact prevention independently PASSED and integrated locally. Further
live alias repairs are held. No new deployment or historical recovery has occurred.
Full report: fact-key-collision-independent-review-2026-09-06.md, SHA
00c585210df9ea757995fd9614839d5d12e616a7b685cf893f9249e740f568ff.

Complete comparison of actual post-stops backup 235134 against fresh 001120
found a lost historical assertion at raw key
`fa:cadformats-file-workbench:status:~in-progress:1788566400000000000`.
Old raw value SHA44805a4cf211693496ac56d89433209a00dfc6e90fc192b0ee4ae2c36ba0e026
became SHA1c7aff7ef471dfb0cf756869db1ba8d398e8a11455fcf4e88a55bd7a4bd02f7e.
The old in-progress fact was already invalidated; the new in_progress fact
has different text and provenance and is current. These literal values share
the same normalized storage slot and validity-start timestamp. This was
intervening ordinary ingestion, not the verified three-key alias repair.

The independent fresh dev-client gate is BLOCKED even though its new Expected
fingerprints match. A matching repair manifest does not certify the source's
earlier fact preservation. Fixed-source technical PASS at 235134 still stands
only for that earlier snapshot.

An unchanged production resolver reproduces the overwrite on synthetic
Velatrix input. The private candidate adds an occupied-key guard inside
PutFact's existing transaction: source, relation, exact destination/value,
raw relation, sentence and start instant must match any existing assertion.
Differences return ErrFactConflict before writes/observer notification.
Exact assertion metadata updates remain supported. The diagnostic reports
only SHA-256 of the key, not sentences or potentially sensitive literal values.

Targeted tests pass for current/history, normalized-value/text/raw-relation
conflicts, complete raw no-write and atomic rollback, same-instant different
zones, and legitimate metadata updates. A full Resolve.Apply collision rolls
back its sentinel entity and preceding valid fact. The real queue worker
parks after one attempt, preserving the original pending text and the old
historical fact. No new retention or provider retry policy was added.

The old merge snapshot-drift test intentionally overwrote text with PutFact.
It now injects corruption through a test-only raw transaction; the full
stale-snapshot refusal assertion is unchanged. This is not weakening the
merge test to accommodate the guard. Missing imports in that private test
adaptation were fixed before the passing test runs.

CGO_ENABLED=0 go test ./... passes in the exact private candidate archive.
Fresh independent baseline disproof, complete restored-source preservation,
compatibility/race checks and benchmark controls also passed. Integrated
combined-artifact and fresh backup/deployment gates remain required. The guard
does not recover the lost row and does not fix earlier canonical-triple
coalescing of different incoming sentences. Neither timestamps nor facts
will be invented to hide the collision.

Private candidate root: `/tmp/scry-fact-collision-fix-sep06.0MxCkU`.
Base source: 226b3e8, without the separate uncommitted tie-order change.
Candidate store.go SHA600a299cf8177ebacafaa80cba2c3778a77545b5bf4f487ba972087e50cfd8a6.
New tests: store/fact_collision_test.go, resolve/fact_collision_test.go and
queue/fact_collision_test.go. Full prospective contract is
COLLISION_PREVENTION_PROPOSAL.md in that private archive.

Separately, a credential-bearing transcript assertion survived the existing
redactor. Its value and derived spelling are intentionally omitted here.
The current redactor handles only PEM, Bearer, ghp_ and sk- shapes; synthetic
ordinary-password and client-secret prose tests reproduce the gap. No real
credential was used, rotated, or removed, and no credential-bearing source
projection is being archived as a review excerpt. Any history cleanup or
credential rotation requires separate explicit authorization.
