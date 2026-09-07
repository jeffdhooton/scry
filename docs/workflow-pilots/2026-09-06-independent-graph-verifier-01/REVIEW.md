# Independent unified code graph verification pilot

Verdict: **PASS for the bounded local graph acceptance**, with the documented potential-call limitation below. No new blocking defect was established. This is not deployment approval or a general call-graph correctness certificate.

Reviewed base: `df598c49bca0784983d136f92c732e2cdacc4561`, with the uncommitted delivery pinned by `docs/code-graph-evidence/2026-09-06/delivery-hashes.json`. Every delivery file and every checks.json log hash matched. `integrity.json` records those comparisons. The verifier did not build or repair this change.

Total review duration was not measured; targeted execution wall time 9.992 seconds, including build/test startup. The fixture cases themselves took 3.41 seconds; the independent non-call and reverse multi-hop probes took 0.48 and 0.12 seconds.

## Required acceptance findings

- **PASS — real source to reopened daemon APIs.** Read both fixture sources, original decoded index evidence, expected IDs/edges, provenance, and the actual harness. Re-executed all four real-index scenarios in an isolated exported checkout: Go/TypeScript, current parse/legacy missing kinds. The harness parses rebased copies of producer bytes, closes the code store, invokes the production graph.build handler, closes registries, creates another handler instance, and uses separate OS client processes for query/path and ordinary code APIs. The fixture graph is not directly seeded.
- **PASS — attributable relationship.** `graph.query {"query":"Speaker","repo":<isolated-root>}` returns one exact symbol with its definition. `graph.path {"from":"Greeter","to":"Speaker","repo":<isolated-root>}` returns the source-backed `implements` relation with exact directed endpoints and definition sites. Go Speaker/Greeter are at fixture.go:3/:7, TypeScript at fixture.ts:1/:5. Go method definition at :9 and TypeScript `implements` clause and method at :5/:6 support the relationship independently of the builder's expected output.
- **PASS — counts and negative evidence.** Re-execution gives Go 7 nodes/2 edges; TypeScript 7 nodes/5 edges. Retained baseline/reversal evidence shows 0/0 for both. Unrelated symbols stay disconnected, absent Go enclosing-scope evidence stays absent, unchanged rebuilds have exact node/edge equality, and another repository has no fixture symbols. Exact RPC requests, responses, and fresh timings are in `targeted-probes.log.txt`.
- **PASS — conservative kinds and consistency within tested cases.** Inspected the pinned binding's parser and fallback; reran malformed descriptors, locals, escaped names, unsupported descriptor types, authoritative kind precedence, and external metadata field ordering. Nodes and call/reference/implementation endpoints use the same classifier. The descriptor Type stays generic and the Go interface method Term is not promoted to a function. Explicit unsupported kinds do not trigger fallback. External endpoints have no invented definition location.
- **PASS — path disproof probe.** An independently authored graph has reversed `references` then `implements` edges between function nodes, plus a dangling side edge. The two-hop A-to-C path retains exact stored labels and original B-to-A/C-to-B directions; the dangling branch is not used. This challenges the previous type-based label inference beyond the existing single-hop tests. This small probe intentionally seeds a graph to isolate path semantics; it is not the producer-fixture proof.
- **PASS based on retained gates — unaffected safety suites.** Inspected hashed uncached full-suite and final-delivery test/vet logs, including code query, git/schema, curated memory, resolver/store/queue tests. Did not rerun full suites merely to repeat their evidence. The serialization build-window tests were rerun and passed. The September 6 reset and August 22 BeginBuild/EndBuild decision were read; no implementation change here touches memory or the serialized build lifecycle.

## Optional finding: `calls` remains a potential-reference label

Location: `internal/graph/builder.go:230` through the confidence assignment at :241; the edge creation existed unconditionally in the baseline builder for already classified symbols.

The independent synthetic SCIP probe supplies source:

```typescript
export function target() { return 1; }
export function holder() { return target; }
```

With missing kind metadata, actual parse/store/build/reopen/daemon `graph.path` returns `holder -> target`, `type: "calls"`, `confidence: 1`. The source returns the function value and does not invoke it. The underlying **reference and endpoint attribution are real**; an interpretation as a proven invocation would be unsupported. The README explicitly documents `calls` as a legacy potential-call label, so this is not a newly discovered blocker for its bounded claim. The fallback broadens the symbols exposed to that inherited convention. A future API-contract change should make this uncertainty machine-visible or use `references` without actual invocation evidence. Do not describe this delivery as proving a precise call graph.

Reproduction: copy `verifier_probe_test.go.txt` to `internal/daemon/verifier_probe_test.go` in an isolated export containing the pinned delivery, then run the command in `targeted-command.json`. `TestVerifierNonCallReference` is a characterization, not an assertion that invocation was proven; its PASS must not be cited as call-graph correctness. Exact source and RPC output are retained at log lines 77–81.

## Coverage limits and test judgment

The real fixture tests are substantially stronger than implementation-shaped unit tests: they pin producer bytes and source and exercise persisted data and production handlers. The classification table tests alone do mirror the mapping, so their green status is not independent proof; source/raw-index comparison and the adversarial path probe provide additional evidence.

This pass did not regenerate producer indexes, invoke providers/network/installers, test production-scale graph latency, fuzz all malformed SCIP data, race arbitrary queries with builds, or independently rerun the full memory suites. Before/reversal evidence was inspected and hash-verified, not regenerated again. FindPath's O(E) edge scan remains an explicitly documented, unbenchmarked large-repository performance consideration. No production correctness or latency conclusion follows from the small fixture timings.

## Reproduction and boundaries

`targeted-command.json` records the exact argv, isolated export root, disabled-download environment, exit code and timing. `export-root.txt` names the preserved temporary export. The exported checkout has the independently authored probe; the shared repository has only this review directory added by this verifier. No production/test changes, live stores, daemon lifecycle, memory writes, installs/network/provider calls, deployment, git push or configuration changes were performed. No finding was repaired.

The baseline revision, full delivery hashes and source fixtures are sufficient to recreate the export. The retained builder reversal procedure and result are at `docs/code-graph-evidence/2026-09-06/reversal-command.json` and `reversal-baseline.log.txt`.
