# Partial action replay and result validation

2026-09-06 after16:35 UTC. Private implementation only, not deployment or approval.
Root workspace `/tmp/scry-complete-admission-sep06.6qYb7K/code`. Full controlling
assertion contract20d5ed6f remains unchanged. Main production e097fa6 and both live
a06cd7b binaries unchanged; no adoption, cleanup, provider operation or live write.

## Original deferred actions, not a whole declaration rerun

Partial declarations now derive a pending-operation program from the actual selected
same-input result. Accepted/rejected operations are carried byte-exact and skipped
before overlay mutation. Remaining metadata/type/alias/removal operations retain
their original IDs, alias ordinals and literals; global decision positions remain
evidence, not cross-retry identity. Current spelling lookup cannot rebind a partial
declaration to a different actor. Historical action evidence pins the original full
EN tuple and exact ig/il/il-consumed selectors even after an earlier retry lost its
binding. Previously retained votes are checked against actual rows before planning.

Only pending type changes run. A type already satisfied by later baseline state
gets explicit no-effect; accepted metadata cannot refill a cleared description.
Ordinary and pending removals share the same closed predicate/effect code. Pending
removal evaluates only its original literal. A later alias requiring removal is not
automatically acquired by an old type operation: that type defers with explicit
new-type-revalidation-target. Original absent removals are explicit no-effect.
Pending duplicate aliases retain separate original ordinals and ordered outcomes.

Unchanged deferred action dispositions, with the same original operation/actor,
no new effect and no changed vote acceptance, retain old immutable action records.
Unchanged complete declaration outcomes retain their original record. Historical
evidence remains historical; this is not permission to execute old effects, and
actual current planning checks still run. A changed disposition gets predecessor
linkage. New action records and original terminal ones compose in original order.

Partial source9b017441fa64c2f270f0b645fef95be956c3a32fb2e30b4f486cce750328ab7c;
test0c973b07cdc628799685b9cb719b4682c7deec44b711563af89a20ca99c22d7e.
Replay source2b0b9102a41b7d7b60e42eb691618ba9f1e9c21aba8a7b092e19b1792626527d;
identity receipts6fa308b430147dff5a8b4b55a68015adb3e807f57e88df4cd9d2fbb694a668da;
shared alias predicate/effect source44741865f641242ad3028d0fbe4efeeb0a46bc8b92da29b723e8da2148830f8f.
There are now two successor overlay files, not the exact tenth reviewed source.

## Retained first failures and fixture corrections

Initial three partial tests FAIL.557s solely on the prior intentional refusal guard,
log88b2632840908cb5f91d92452e8b0b8343c355de6ff1ad7d9096d14335616840.
First test8be4fd1fb47e89353a7d6f26a3ffc499b4d96d6df711b753940b817e6b7b75e5 and prior
replay5514b8e are preserved unchanged. The old temporary refusal testac69b170 is
also preserved as partial-old-refusal-guard-test.go.txt and in frozen khrGn1. Its
same original fixture now requires successful byte-exact no-op, not refusal. This
is the intended implementation advance, not a weaker preservation assertion.

The first implementation passed two cases but failed the claimed repaired-claim
case, log3a8483b51e91cd2e1b68165b15e75195d8cef7eda903c8f14f3397c31e847fb3.
Diagnosticfc3f543766d90752ec14a6a178ceaa720ed4184a4bbdec5f8bdcbaa35ada3ec1 proved the
fixture had only called PutEntity; that intentionally does NOT refill an existing
missing alias claim. The fixture now seeds and asserts the explicitly stated exact
repair. No production policy changed to make that test pass. Corrected partial,
root terminal and unchanged independent terminal tests PASS1.402s, log
5651ebaf6cbc93c131fa6d6f61401f7e6ed45a816442b9982d7b716bc27d6c27.

Expanded original alias ordinal/duplicate cases passed in both orders. The new
later-removal fixture incorrectly expected retained policy to remove frontend;
FAIL log8f812e97b687ad768ef7963d6c470580a32d8a4656a90c3949fccbee1e90b57e and test06ced102
are preserved. Diagnostic489a9ba58e2aeaabf1b1f09f1eacf22fe82f5ba454f2e58dbd6f64e555fb5569
proved the ordinary program proposed no removal. This repeated a known earlier
fixture pitfall; it is not a newly fixed policy bug. The case now uses the retained
Atlas box counterexample and first asserts an actual ordinary removal proposal.
No assertion or policy weakening. Expanded cases PASS.537s, log
0f7f0adbe64b1e46dea4f72c8692adc7385488e2e99cbf98b7149104d2cdec57.

Additional original pending removal/present-or-absent cases and repeated successor
selection pass. New effects are synthetically seeded separately from terminal
effects; the fixture never reapplies carried metadata. Lost spelling moved to a
different real actor remains a stable partial retry after actual successor head
selection. Entire partial suite PASS.812s, log
b7f24d3281a53f958001c604eb34af82bcfd1c34f086fb2f9d30842b6156b060.

First full uncached no-CGO suite after partial replay PASS store76.462s/
resolve16.964s/daemon27.596s; completed log
0341ae5bf1904497b617be28bbe12f03b6cd1242afdbbe8c4acee443decfba79; vet PASS.
These runs used the OS default test temporary directory, unlike earlier private
runs. Subsequent commands explicitly restore the private test-tmp setting. No
actual memory store was used. Root diagnostics remain test-only, not production.

## Action, birth and component semantic validation

Twelve new action corruption cases all initially FAIL.479s: unknown operation,
changed proposal actor/alias, collateral effect name/description/type/alias,
wrong claim key/owner, stolen claim before-image, missing effect before-image,
invented metadata vote. Original testd180001d/source1e78b836 preserved; log
6c509cbac7f2765a014c8dcef8a31a33cec08c63d42dcea44ed1fe35e881eeef.

Source-only validation binds the full proposal to its exact decision in owned
evidence, original input action ID/role and disposition. It recomputes permitted
metadata/type/birth/alias/removal deltas from the recorded filtered before-image,
checks stable identity, exact claim key/owner, and forbids alias transfers and
collateral metadata changes. Vote flags are restricted to alias operations, with
explicit predecessor handling for earlier retained votes. It validates stored
records, not authority to rerun them against current state. Unchanged twelve cases
plus partial/root/independent replay and identity receipts PASS2.090s, log
775fb12b09c0e3c7620d2d10361a69774547bcab95a2096d136077d0a3cc8824.

Two further birth cases (another actual input observation; wrong original role)
and asymmetric component membership initially FAIL.520s, log
eef68adcd9b81a42cbcdf4d24b9008867143507104e96f4a7656312e9f846a69.
First sourcecd1eb9d5/testd5df1354 preserved. Source-only correction reconstructs the
full original first birth tuple and requires reciprocal complete component sets.
All selected admission and fresh replay/preparation groups PASS5.253s, log
a36697cfd6cbb6f086af35d3d318938f241ff4c7d5d9cbf3415a6f2069383481.

Current action validator dba1402e0038a12280f3ad3ad0cf7f8b51870da36790f75917ad745643bcddfe;
result b2da63627e3b4422ff43b8c137e0bf21f5dc53cd7f394386923986ad4034da21.
Expanded testd5df13548e5990cb747d61b4e677a5dbe1769c9a71197af9fa1c46683c309558.
Later full suite is running; its result must be recorded separately on completion.

## Not yet delivered

No actual complete materializer or result/head/conflict writer, fixed final actual
inventory checker, normal Apply adapter, policy deduplication or all-writer/schema
floor exists yet. Synthetic successor seeding is not a production write test.
Retained vote with deferred effect and every granular action interaction still need
expanded actual-program coverage. Explicit repair lineage, full hint eligibility/
supplier semantic validation and generation-prefix corruption cases remain open.
No new independent approval, candidate actual-store probe, live adoption, deployment,
repair, final grade or goal completion. Latest actual backup155247/note276f closure
and last candidate probe132039 remain as previously recorded.

## Completed full validation run

Full uncached no-CGO suite after action/birth/component validation PASS store70.257s/
resolve16.888s/daemon29.735s; final log
31184c34b674a92180e827b08e46457f992d2d8357e6c818c62d4feb2904008f.
Relevant vet PASS. This supersedes only the running-test status above; it is root
evidence, not independent approval or a delivered write-path certificate.
