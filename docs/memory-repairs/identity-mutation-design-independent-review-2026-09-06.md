# Identity mutation design — independent bounded review

2026-09-06. Reviewed design SHA-256 `aede57c793ba2468e17e1baba0758ce3ea5079325741b346e5127c9a473617d2`, baseline commit `099585964fe6d049520583358f491959b0ba2dcb`.

**GO for a private, uncalled en:/al: actual-writer ledger with the conditions below. NO-GO for production routing, support selection, provisional cleanup, legacy adoption, or controller integration on this design alone.** I found no contradiction making the proposed complete undo plan impossible. The baseline-defect exception can be safe, but only as equality of complete ownership relationships, including absent entities and every retained listing. Alias-key bytes or an unchanged entity slug alone do not establish that equality. The draft needs the explicit closure and precedence rules below before its exception becomes an executable finalizer policy.

I ran `scry memory orient --cwd .`, read the complete active objective, complete pinned design, and complete archived ControllerV3 review. I inspected owner, journal, PutEntity, alias helpers, DropAliasRehome, DeleteEntity, raw-reference inventory and legacy anchor/adoption sources. I exported baseline into `/tmp/scry-identity-design-review.QcQ7fg`. Only a new synthetic characterization test and this report were authored in that private export. No candidate/shared changes, live services, private live replica inspection, real data content, providers, config, sweeps, rooms, remember, deployment, or other external writes occurred. This is design disproof, not an implementation or whole-goal grade.

## Executed baseline characterizations

`go test ./internal/memory/store -run TestIdentityMutationDesign -count=1 -v`

PASS: five synthetic tests; package completed in 0.585s. PASS means the stated baseline behavior was reproduced, including the undesirable behavior. No new production primitive was implemented or tested.

1. `EarlierBirthUndoErasesLaterOwner`: capture absent `al:shared`; Atlas creates and drops shared; Borealis creates and lists it; exact journal restoration deletes Borealis's claim. Borealis still lists shared. The journal correctly restores bytes but has no actor policy. The proposed rule retaining the validated final Borealis owner defeats this counterexample. Never choose undo keys merely because Atlas once touched them.

2. `DropCanonicalDuplicateCreatesHole`: Borealis has canonical Name `Shared` and alias `shared`. DropAlias removes the alias and owned index row while leaving canonical Name `Shared`. It succeeds. Thus a correctly attributed retained actor's delete, even through an existing helper, is insufficient proof that preserving the deletion is safe. The final relationship check must include canonical names, not just Aliases. Smallest correction is to refuse this final graph; do not silently rehome or infer a winner. A later explicitly reviewed name disposition could authorize a different result.

3. `StaleBeforeImageRestoresOnlyBytes`: ClaimAlias accepts a missing target. After a subsequent claim and exact restoration, the old alias bytes again point to `absent-old-owner`, whose en: record remains absent. This establishes the raw preservation case and also demonstrates that low-level ClaimAlias success is no recognized-identity certificate. Exact restoration can preserve this old defect only when every relevant baseline relationship remains unchanged. It must not create an entity, select a generation, or inherit old claim/attestation authority.

4. `DeleteLeavesUnlistedClaim`: Atlas has an additional ClaimAlias-created key not present in its name/alias list. DeleteEntity deletes Atlas but leaves that key pointing to it. The helper's comment promises every al: key, but the implementation traverses only name/aliases and normalized slug. Therefore ordinary DeleteEntity is not a complete cleanup primitive. Retain the proposed full raw alias inventory; reject or separately instrument/authorize this helper in admitted scope.

5. `UnchangedAliasBecomesDangling`: Atlas owns `al:shared` without listing it; retained Borealis lists shared as a legacy conflict. DeleteEntity removes Atlas but leaves shared's exact bytes and Borealis's listing unchanged. The defect changes from an owner with an en: record to a dangling owner even though the alias key and retained listing are byte-identical. This directly disproves a reduced exception based on those two fields. It does not disprove the draft's stronger recognized-identity relationship requirement. That requirement must actually inspect identity state for index targets even when the alias key is untouched.

The fixtures use descriptive identities and baseline APIs; they make no factual-support or recognized-generation claims.

## Smallest necessary precision for the eventual exception

Define the relationship closure explicitly. Start with all changed en:/al: keys and every name/alias listing added, removed or changed by entity writes or proposed undo. Include aliases whose raw owner is any changed/removed identity even when no entity lists that alias. Include every baseline and proposed-final entity listing each affected normalized key. Include the exact owner bytes or absence, exact listing occurrence/spelling and whether it is a canonical name or alias, and the identity/selector/consumption state for every listing identity and index target. Natural-slug routing, where separately recognized by the resolver, needs its own stated coverage; do not silently equate slug routing with the current PutEntity name/alias set.

Full raw en:/al: inventories can derive this closure without a maintained reverse index. The digest detects final drift; it does not itself enumerate relationships or select semantic ownership. A list/key scan must compare its expected projection to the actual final projection after undo. A changed identity that makes an untouched index target dangling must enter the closure. Changes of lifecycle identity at the same textual slug must enter it as well.

Use explicit precedence: a final defect is permitted only if its complete relevant relationship equals its pinned defective baseline relationship. Otherwise the final relationship requires independent valid ownership and approved listing/disposition proof, or the whole transaction refuses. This preservation exception overrides the draft's otherwise unconditional retained-final-owner validation only for that exact unchanged defect. It never upgrades the defect into valid ownership or supports new births.

In particular, baseline `al:k -> absent C` may be restored when C remains absent and all retained listing relationships are unchanged. If retained B newly lists k, restoring that stale row is no longer the same relationship and must refuse. Baseline `B lists k; al:k absent` may remain absent across an unrelated description update, but newly creating that hole or adding a new spelling under an old normalized key cannot use a blanket “B was preexisting” exception. Exact original bytes remain available even where strict decoding cannot establish a relationship; unknown data must lead to refusal where proof is required, never lossy reconstruction or assumed ownership.

Writer attribution remains separate from operation authorization. An al: put actor matching its new owner and a delete actor matching its current owner provides a useful mechanical invariant. It does not prove a transfer, deletion, lifecycle transition, or listing was reviewed. Ordered history should have a stable entry/operation ordinal to which the fixed owner's later authorization evidence can be bound. If no such binding is available, an entry cannot satisfy “that operation's own proof.” A support boolean, caller-selected cleanup slug list, or successful ClaimAlias/DropAlias return is not that proof.

## GO conditions for the next private unit

Implement only private actual Set/Delete methods scoped to one admission owner and body phase, for en:/al:. Keep owned exact key/before/after bytes and ordered actor entries; capture through the existing journal before mutation; validate actual current state against the last recorded after-image. Do not broaden the journal's att: support into this new ledger's allowlist. Empty/invalid/noncanonical keys and new entity records must refuse. Deleting an absent key must have an explicit no-op/refusal contract and cannot fabricate an owner-attributed deletion. Existing opaque before-images remain exact bytes.

Make every primitive error poison the owner, including validation before entering an update callback and caught errors. Clone caller-provided key/value buffers before Badger retains them. Keep history readout copies isolated from caller mutation. Refuse use during finalizing/closed phases and across owner/transaction boundaries. Freeze any eventual finalization plan before performing undo; private journal restoration belongs to a separate fixed finalizer path rather than pretending it is an ordinary body mutation.

For this smallest unit, accept no support decision, undo plan, generation/legacy selector, or reviewed ownership transfer supplied by callers. Do not route existing production writers yet, since that would change ordinary behavior and require the complete authorization policy. Test the private writer's mechanical contract independently: multi-actor same-key sequence, put/delete/no-op history, malformed/mismatched actors and bytes, recapture preserving first image, caller buffer mutation, swallowed failures, panic/rollback, closed owner, transaction mismatch, and stale actual after-image. This is a finite primitive review bar.

Production integration must additionally audit all direct mutation sites, including `merge.go`, `retire.go` and `unalias.go`, or prove their admission refusal; routing just the listed store.go helpers is not an all-writer proof. Full before/final inventories catch untracked final differences, but cannot prove absence of transient untracked writes restored before the final scan. Lifecycle checks and all actual writer routing are still needed. Public API validation failures outside `update` also need poisoning once those APIs are admitted. Arbitrary concurrent raw-writer phantoms remain outside the local-ledger proof.

No support-policy, dependency closure, immutable outcome/current projection, B evidence, raw fact ledger, lifecycle adoption, cleanup, prevention deployment, performance, recall, real-sweep or whole-goal clause is certified here. Existing finalizer injection is a private harness, not a completed fixed policy.

## Input and evidence pins

Relative paths below refer to the private export.

| Input | SHA-256 |
| --- | --- |
| Pinned design | `aede57c793ba2468e17e1baba0758ce3ea5079325741b346e5127c9a473617d2` |
| Archived ControllerV3 review | `2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10` |
| `internal/memory/store/store.go` | `9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891` |
| `internal/memory/store/identity_admission_owner.go` | `14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82` |
| `internal/memory/store/identity_journal.go` | `d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80` |
| `internal/memory/store/identity_reference_inventory.go` | `d80563bb046ef9721bf42ba2199d6207f38f7629b2c9196a129accefa24877d4` |
| `internal/memory/store/identity_legacy_anchor.go` | `7a84b00d46977c2d8ce97c5b29f35763fa347bcd61f967d7178ef42a399a7de7` |
| `internal/memory/store/identity_mutation_design_characterization_test.go` | `897d7f20bc9ab5606124117cb9819c89a7c3bf275ef0676eab949ed344005bda` |

This report's hash is supplied separately to avoid self-hash mismatch.
