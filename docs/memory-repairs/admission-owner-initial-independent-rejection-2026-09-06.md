# Private admission owner harness: independent rejection

2026-09-06. Verdict: **NO-GO for the exact frozen candidate.** Its ordinary
update/AtomicWrite control boundary survives the independent disproof cases, but
the claimed post-return facade refusal and rollback boundary are false for public
maintenance methods on the very same facade. No production controller, support
policy, observation policy, cleanup, generation adoption or whole-goal result is
certified here.

I ran the required initial `scry memory orient --cwd .` and read its output, the
complete active goal, complete OWNER_CONTRACT.md, and complete archived Controller
V2 review. I exported base 87a6d1a262d5d7c62037d66335b9a020f89fd4a6 into this fresh
private directory and copied the three exact candidate files with apply_patch.
All candidate hashes match the provided frozen source. I changed no shared or
frozen source, implemented no fixes, and accessed no live graph, providers,
sweeps, deployments or configuration. The initial orientation was the only memory
interaction. All fixtures use fresh temporary Badger databases.

## Proven counterexample

`Store.MergeEntitiesChecked` at internal/memory/store/merge.go:125 opens
`s.db.Update` directly at line 132. It does not consult the admission owner or its
phase, and its maintenance mutex belongs to the facade rather than the root.
Consequently exposing the full *Store facade retains a public independent
transaction capability.

The independent fixture creates two synthetic tool identities plus one attribute
fact. It obtains a real ready merge preview, pins its expected fingerprints and
metadata, and explicitly selects the synthetic disposition. It does not infer an
owner, supply a raw transaction, or write through a separately captured root Store
inside the tested closure.

Two executions prove the problem:

1. Run admission successfully and retain its facade. The owner is admissionClosed.
   Calling escaped.MergeEntities(request) succeeds and commits the merge, deleting
   the loser and moving its assertion after the admission wrapper returned.
2. Call tx.MergeEntities(request) in the injected finalizer and then return a
   deliberate error. The wrapper reports that error, but the merge already committed
   its separate transaction; complete raw-state comparison detects the change and
   the loser remains deleted.

The second test uses the private finalizer only as the authorized harness seam.
It does not claim a future complete store-owned policy already exists. The first
test independently disproves the current explicit post-return facade claim,
without any supposition about future policy.

The contract excludes separately opened root-store writes and arbitrary raw
transaction misuse. These tests use neither: the write is the public MergeEntities
method of the supplied admission facade. Therefore the counterexample cannot be
explained by those exclusions. The ordinary update/AtomicWrite-only freeze is
insufficient to justify the broader facade safety acceptance condition.

`owner_maintenance_boundary_disproof_test.go` characterizes the bypass and PASSES
only when both unsafe behaviors are reproduced. Separately,
`owner_maintenance_required_safety_test.go` asserts the required safe result and
FAILS both cases on the frozen candidate. This latter file can be reused unchanged
against a correction, together with the ownerRaw helper in
owner_independent_disproof_test.go. Exact failure output:

```
admission facade accepted public maintenance at after-return
public facade maintenance changed committed bytes at after-return
admission facade accepted public maintenance at finalizer-failure
public facade maintenance changed committed bytes at finalizer-failure
```

## Checks that did survive disproof

Independent tests assert a complete raw key/value map, including an opaque unknown
family, and zero observer events after ordinary admission failures. They cover
entity records, alias indexes, an episode, a fact and reverse adjacency. They also
verify two nested AtomicWrite levels share the same facade; a body write after the
nested callbacks is visible to the single final check; no events are emitted
before commit; the owner closes before observers run; and expired ordinary write
and callback attempts refuse.

The passing error matrix covers body error/panic, caught nested callback
error/panic, recursive admission, finalizer error/panic, ignored finalizer
PutEpisode/DeleteEntity/AtomicWrite refusal, and a real Badger oversized-key
staging error reached through PutEpisode. Poisoned bodies never invoke the final
check. Finalizer cases invoke it exactly once. Admission entry inside an ordinary
parent, caught two levels up, still poisons the parent's commit. A normal parent
with an ordinary caught nested error still commits, preserving baseline behavior.

A separate synthetic competitor invalidates a real Badger read set, producing
badger.ErrConflict at the owner's actual commit. The test verifies precisely the
competitor's intended bytes remain, none of the owner's mutations leak, zero owner
events occur, and the facade closes. This explicitly scoped competing raw
transaction is fault injection for a commit error; it is not used as a claimed
confinement counterexample.

These passing tests neither certify arbitrary raw-transaction/concurrent facade
use nor public validation failures occurring before update. Those are explicit
contract exclusions. They also do not certify complete facade cache disposal or
any production integration.

## Commands and evidence

All commands run in /tmp/scry-owner-independent-sep06.lWNWgh with CGO_ENABLED=0
and `-count=1`.

- `go test ./internal/memory/store -run 'TestOwnerIndependent|TestAdmissionOwner|TestAtomicWrite' -count=1 -v`:
  PASS; owner-targeted.log. Four new independent top-level tests, the four original
  harness tests and the unchanged ordinary AtomicWrite test pass.
- `go test ./... -count=1`: initial full suite PASS; owner-fullsuite.log. This run
  includes the independent ordinary boundary tests and precedes the newly added
  public-maintenance counterexample and safety regression.
- `go test ./internal/memory/store -run TestOwnerMaintenanceBoundaryCharacterization -count=1 -v`:
  PASS reproducing both unsafe cases; owner-maintenance-boundary-final.log.
- `go test ./internal/memory/store -run TestOwnerMaintenanceRequiredSafety -count=1 -v`:
  FAIL, exit 1, both required safety subcases; owner-required-safety.log.
- `go test ./... -count=1`: final full suite including all independent tests FAILS
  with exit 1 only in the two required maintenance safety subcases;
  owner-fullsuite-with-disproof.log. All other packages pass.

An initial maintenance fixture lacked facts and was correctly refused as leaving
a hollow survivor. Its failed fixture-construction log is retained as
owner-maintenance-boundary.log. Adding one synthetic attribute assertion made the
reviewed merge valid; the final characterization and regression logs above are
the actual candidate evidence. No candidate implementation changed.

## Pins

SHA-256, relative paths unless specified:

| Artifact | SHA-256 |
| --- | --- |
| Frozen OWNER_CONTRACT.md | e47d67226d6393cfea527ba41ab14ae11389cac16af8df7912a7206c1bb223cd |
| Archived Controller V2 review | 67218c1c362ab20f8a0d328f3a1895935b132c76c33c1e31c5a8f955a3fde49b |
| internal/memory/store/store.go | 1fd30285405a1a2ee11d7213c84488f6c227d7737f21e20c8e468389935f6fc1 |
| internal/memory/store/identity_admission_owner.go | a4d48fc65f87b3723ac11a846679c0e2abf4b8f2dfa1d5e5ab90d688ca632cdb |
| internal/memory/store/identity_admission_owner_test.go | 903298415d0d4a2e65fcbf6864d89fd91f0118d58c73447f33aed6ef3341d449 |
| internal/memory/store/owner_independent_disproof_test.go | 71e2acf902d5c6f16f3941a4135275b222271056f797d369b0edc9c7a00d0d31 |
| internal/memory/store/owner_maintenance_boundary_disproof_test.go | f282914916ee0c258e3444606a3c747178bf33430bc013b1c87cf5d42aa664dc |
| internal/memory/store/owner_maintenance_required_safety_test.go | c8d442b59912e49ef76e279332bb882f80f0f3fae8ff9f0f3e910118226f695c |
| owner-targeted.log | 4f27b1b3b0e926a12ee851321966d02f5027f836914d6dda28b65f022f32171c |
| owner-fullsuite.log | f72047d48051f8b24fbf5d5fdbac81d432c2a26d980e5a6cb79ae26f73d1d7d2 |
| owner-maintenance-boundary-final.log | 04774b95e6774ac0cbd14c65567117ab5c2d868230438377b1c6b9b85bdd005e |
| owner-required-safety.log | 3511f2257d0f9959dbc31777480b8a04cf981608c6b7e0045289decbf61304ce |
| owner-fullsuite-with-disproof.log | 0c93caa672ead9d9fff3ad64acb696b4ed444dd50938bc47629f656265ade257 |

The root builder has acknowledged the counterexample and intends to refuse public
independent-transaction maintenance through admission facades. That is a proposed
correction, not reviewed evidence. All such changes require a new exact candidate
and independent review; this rejection remains attached to the frozen hashes.
