# PASS — actual live six-record migration merge, bounded post-apply audit

The actual automatic pre-apply and immediate post-apply backups reproduce exactly the reviewed six-record manifest's predicted changes. No unexpected raw key/value change occurred. This is an actual-backup verification, not merely a replay of the builder's replica result. It does not grade recall suites, later ingestion, two-sweep convergence, or graphwide completion.

Pinned evidence:

- Actual automatic pre-apply backup `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200709Z.badger`, 77,827,994 bytes; SHA-256 `acccdd658024d440eb98e0373cfb303af561bb370ae6a49998d3d28fe2a09744`.
- Actual immediate post-apply backup `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger`, 77,829,763 bytes; SHA-256 `5323fef7b594c748fd61462909cc808f78c1bd0a104da070abdc54cb21e102aa`.
- Reviewed source `/tmp/scry-migration0160-fresh-sep05.teYtyC/source.badger`; SHA-256 `7261cfff0a7957d825855c430dbbc0f333517e806b7c47d8fc03a66fe8fc1548`.
- Exact manifest `/tmp/scry-migration0160-fresh-sep05.teYtyC/six-member-preserved-refs/manifest-replica-only.json`; SHA-256 `547b324b4a0c83bcec6a345b46a8e2f3e80be5b2178b002d125d2aaf8f29e2c3`. Its committed copy was independently verified at `a94bf9f` in the pre-live report.
- Implementation independently extracted again: `393eeec79f80d3b4becff276c4fcffd71fa68ac5`.

I restored all three source files into new reviewer-owned stores under this directory. Every actual pre-apply raw key/value equals the reviewed source except exactly `meta:last_sweep_at` and `meta:last_sweep_report`. The complete keyset, including empty adjacency values, was independently checked. Thus all entity metadata, current/history facts, episodes, alias claims, and the full six-member semantic closure remain exactly the reviewed inputs. Previewing the manifest on the actual pre-apply restore is ready and reproduces every expected fingerprint without writing.

I constructed the expected post-state directly from the actual pre-state and explicit manifest: retire exactly the five listed loser keys, write the reviewed survivor metadata, route the complete eight-key approved spelling set to the survivor, and relocate only the nine touching facts and their reverse edges. Comparing every raw key/value against the actual post-apply restore yields zero differences from that prediction. Exactly 35 keys change; total keys 241,340 → 241,337. No outside metadata, aliases, control identity, episode, cursor, or operational metadata changes occur between the actual pre/post backups.

Actual result:

- All 80,203 facts preserved: 72,411 current and 7,792 invalidated. Seven group facts relocate endpoints; two existing survivor facts remain unchanged. No fact text, raw relation, value, validity, confidence, or provenance changes. Nine touching facts remain current.
- Entities 30,441 → 30,436; all five named retirees are absent. No alias owner or fact endpoint still references a retiree. Survivor metadata exactly matches the reviewed canonical qualified filename, tool type, both repository associations, earliest creation, latest observation, and complete approved aliases.
- All 9,321 episodes are unchanged, including the episode that limits earlier merge assertions to replica/proposal history.
- Every approved spelling resolves to the survivor and returns nine facts with history included. Independently tested twelve literal/normalized variants cover all eight normalized keys, including `Docket migration 0160`, the qualified SQL path, and former hollow slug.
- Cross-type collisions independently measured 484 → 482. Complete hollow inventory decreases only by `dbmigrations0160-task-evidence-rulessql`, 2,855 → 2,854. Complete dangling-endpoint and self-loop inventories remain unchanged (994 distinct dangling endpoint slugs and 1,095 self-loop facts globally). No new group hollow, dangling endpoint, or self-loop appears.
- The actual automatic pre-apply backup restores successfully and reproduces the reviewed pre-state, establishing a usable rollback source. No live rollback was performed.
- Preview on the actual post-apply restore reports exactly the five absent retirees and is not ready. It changes no raw state. A repeat apply attempt on that isolated restore rejects and changes no raw state.

The table, API modules, reservation coordination/range, encompassing implementation task, repair-group concept, and review artifacts stay separate. Their preservation is established by complete actual raw-state equality outside the explicit merge changes, with their semantic distinction documented in the preceding fresh-source reports.

Checks passed: `TestIndependent0160ActualLive`, `TestIndependent0160ActualPreCompleteKeyset`, and `TestIndependent0160ActualInventory`. Evidence: `results/actual-summary.json`, `results/unexpected-raw-delta.json` (no entries), `results/actual-pre-keyset-proof.json`, `results/actual-pre-preview.json`, `results/actual-second-preview.json`, and `results/actual-inventory.json`. Review code is confined to this independent archive.

No shared or live store was opened or mutated by this reviewer; all restores, previews, and the rejected repeat apply used new temporary reviewer-owned stores. Bounded actual-live verdict: PASS for the six-record merge reflected in these exact automatic pre/post backups. This verdict supplies no approval for further live graph writes or later state.
