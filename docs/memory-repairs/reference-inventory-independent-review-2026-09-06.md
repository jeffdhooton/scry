# Independent reference inventory disproof, 2026-09-06

Bounded verdict: PASS; no contract violation proven. This verdict covers only the private uncalled complete point-in-time raw fact endpoint count/digest scan. It does not approve identity ownership, admission, support selection, provenance closure, fact-writer attribution, dependency planning, concurrent raw-writer phantom immunity, deployment, live graph repair or completion of the memory-solid objective.

I ran orientation, read the active objective, full candidate contract, source, supplied tests, existing strict raw decoder/requested-slug checker and transaction guard. I exported commit 315fa2a24518995bc62ca54d3cc47f0bac17e48c to the fresh private directory /tmp/scry-reference-inventory-disproof.5q0TdQ and copied the three exact candidate additions using apply_patch. Decoder and generation implementation match the candidate byte-for-byte. No candidate source was changed.

Hashes (SHA256):
- REFERENCE_INVENTORY_CONTRACT.md: fb1ca6fd6aacae436eda2e2dc8d51c9900518452df2a8a5809caa27bb114dbf8
- internal/memory/store/identity_reference_inventory.go: d80563bb046ef9721bf42ba2199d6207f38f7629b2c9196a129accefa24877d4
- internal/memory/store/identity_reference_inventory_test.go: e356d5cc9d1160c15504972d58a21a9ef189fdaa825e46f0334a700ee4daf6f5
- internal/memory/store/inventory_independent_disproof_test.go: 54ef0948a3e536bdac48f0d69b34b53181d39843cad48cef05aabde3c00ca9da
- internal/memory/store/identity_fact_references.go: c28aed0f6debabbd1a9d1ff4608b7a88bb4129e59e3eea341b7603d6b6aa3e3e

Independent synthetic disproofs:
- 123 raw facts, inserted in reverse order, with repeated source/destination identities, current and historical edges, self-loops and attributes. No entity, adjacency or retained episode closure is supplied. Counts computed independently from fixture endpoints match every entry.
- Digest assembled independently using explicit eight-byte unsigned big-endian shifts for each key and raw value, byte-sorted keys, exact domain and NUL prefix. Includes opaque duplicate unknown fields, an oversized unknown number and a 70,001-character extension. Every retained raw byte participates.
- Empty/non-fact-only stores produce equal domain-only hashes. Opaque malformed non-fact values are ignored.
- Sentence, raw relation, confidence, provenance, whitespace and opaque-extension-only changes leave counts unchanged and produce distinct independently verified digests. Restoring exact raw bytes restores the full report.
- Known-field duplicates, including case-fold equivalents for all ten decoder fields, malformed unrelated rows, trailing JSON, invalid UTF-8, key mismatch and unknown unmatched surrogate refuse the entire report. Refusals return zero report, match the original requested-slug checker and disclose no canary content.
- Original requested-slug checker parity includes duplicated requested slugs and a zero-reference absent slug.
- Caller mutation of one returned map does not affect subsequent scans.
- Same-transaction staged invalidation, insertion and deletion are observed by both counts and the independently recomputed digest. Successful commit and forced rollback behave correctly.
- Both committed and aborted escaped facades, and a closed base database, return static refusal with zero report. Supplied tests also cover nil facade.
- Exact raw row equality, observer event absence and pending-event equality hold around scans and refusals.

Commands and results:
- CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentInventory|TestReferenceInventory' -count=1: PASS.
- After the final independent raw-only-change test addition, CGO_ENABLED=0 go test ./internal/memory/store -run '^TestIndependentInventory' -count=1: PASS (0.487s).
- Final CGO_ENABLED=0 go test ./... -count=1: PASS, exit 0. Store 27.198s, resolve 14.742s, daemon 27.655s. No failing disproof was found or removed.

Static inspection confirms only test call sites exist, the implementation returns counts and a digest rather than a retained Fact slice/raw row map, and no mutation/event APIs are called.

This grader did not restore or measure the actual snapshot. The root's separate restored051519 observation is not independent evidence from this review. No real data was printed; no live service, provider, configuration, room, deployment, remember or shared-worktree mutation was performed. All authored files and test artifacts are confined to this private export. The existing global objective remains open.
