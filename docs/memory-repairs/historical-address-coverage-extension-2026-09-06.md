# Independent test-only coverage extension

Verdict: **PASS for the test/documentation-only follow-up.** The actual fallback self-supersession fixture passes, and all production bytes remain identical to the independently reviewed revision. This grants no new integration, deployment, live-store or whole-goal certification; fresh replica compatibility remains pending.

Compared the complete frozen candidate directory `/tmp/scry-historical-address-sep06.Mbi8Rn` with the previous untouched suite copy `/tmp/scry-historical-extension-suite.CaPwrh` using `diff -rq`. Exactly two files differ: `internal/memory/resolve/historical_test.go` and `HISTORICAL_ADDRESS_PROPOSAL.md`. Read both complete diffs. No production or preexisting test changes are present.

The test adds `aliases_index_to` while retaining `uses` and `measured`. Its explicit checks require the actual stored fallback Relation, original RawRelation, and entity Dst `caldera`. Existing checks still require original start, historical InvalidAt at episode time, .95 maximum confidence, deduplicated episode provenance, and one addition/one merge/one invalidation. The proposal accurately corrects measured-on-empty-store to a status attribute and references the pinned independent extension `e61342b24da1741cfc66b6aac70d4bd37295e135be5c12cc6461205fa0907e6f`.

Created a separate private copy `/tmp/scry-historical-coverage.XK7neC` and ran:

```sh
CGO_ENABLED=0 go test ./internal/memory/resolve -run '^TestHistoricalAddressSelfSupersessionStaysClosed$' -count=1 -v
```

Exit 0, package 0.370s. All three subcases passed: uses, measured and aliases_index_to. A final `diff -rq` found the review copy identical to the frozen candidate before adding this report. No broader matrix/full suite was rerun for this small extension; the previous independent full-suite and adversarial evidence remain in force for unchanged production.

Before/after SHA-256 verification:

| Relative path | SHA-256 |
| --- | --- |
| internal/memory/resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| internal/memory/resolve/historical.go | 9749aef36ee99197b2b57464cb9b12c7ebd2752d1f8037b7fdade591a4bafb4f |
| internal/memory/store/historical.go | 896ba4d7739eded14bf30df9d5f5afa23c6e2a85f86917f9e495af4dc14c7de6 |
| internal/memory/store/historical_test.go | 04773bd70e86428d94419d28bbda6505a4ac47c0126508da2d2752dfdc285742 |
| internal/memory/resolve/historical_test.go | c9e8d5d2ac411ad45e72fa42c63477b71cbfb347b22dcb346941bf204b5d696e |

The first four pins are unchanged; the final pin is the reviewed test-only change. Original reports, test evidence and baseline copies were not modified. No shared source, live store, real queue, external provider, configuration, credential, backup, integration or deployment was accessed or changed. All tests used fabricated temporary data; only this private report was written with apply_patch.
