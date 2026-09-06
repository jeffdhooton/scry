# Controller V2: independent bounded design review

2026-09-06. Verdict: **GO for a private, uncalled immutable occurrence-observation primitive. NO-GO for production controller integration or adoption on this draft.** The draft is substantially compatible with the earlier reviews. Its explicitly unresolved generation/adoption mechanisms are open design choices, not reproduced implementation defects. No controller implementation exists here and no whole-goal certification follows.

I read session orientation, the complete active goal, complete CONTROLLER_V2, the full initial and delayed unattached-admission reviews, the fixed journal review, and the initial generation rejection. I inspected the exact private generation correction but do not supersede its separate review. I independently exported baseline a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897 into this fresh private directory. Only a new characterization test and this report were authored; original production sources/tests remain unchanged. No shared checkout edits, live graph access/writes, provider calls, retries, sweeps, config changes or deployment occurred. The required initial memory orientation is the only memory interaction.

## Concrete evidence and necessary corrections

### 1. Finalization belongs to the outermost Badger owner

At store/store.go:236, nested AtomicWrite directly invokes fn(s). It creates neither a new commit nor a boundary after which the enclosing caller cannot write. TestControllerV2NestedScopeIsNotCommitBoundary independently reproduces this: an inner check returns, the outer callback creates a zero-fact node, and the outer transaction commits it. This is baseline behavior, not a claim that an absent controller failed a test.

V2 correctly requires unavoidable store-owned finalization but must explicitly cover admission entered from an already active ordinary AtomicWrite. Wrapping ApplyWith's callback alone is insufficient. The safest first wrapper contract is: refuse admission entry on an already active non-admission facade; inside an admission-owned outer transaction, nested ordinary AtomicWrite reuses the owner and finalization runs once, after its outer callback, immediately before Badger commits. Alternatively extend AtomicWrite itself with outer-owner finalization state; that is a larger unit and needs an explicit case where Apply returns to a caller that writes again. No exported callback may set completed/finalized state. Finalization errors must poison/refuse the transaction even if a nested caller discards the returned error. An early finishSupported, nested success, error caught by the outer callback, panic, and writes attempted after finalization are separate tests. Preserve facade cache disposal on every exit and zero public Stats for returned failures.

### 2. A decoded AllFacts slice is not a validated support oracle

At store/store.go:891, AllFacts scans the actual fa: family, not the draft's f: spelling, and calls json.Unmarshal without checking key/body agreement or duplicate JSON members. TestControllerV2AllFactsDoesNotValidateRawEndpointAuthority supplies two synthetic raw records: one contains duplicate dst members and the other has an address naming dangling but a body naming other. Both decode successfully as other while all original bytes remain stored. A decoded projection can therefore hide an ambiguous reference before a destructive cleanup decision. This confirms V2's requirement to refuse ambiguous raw records and makes its implementation obligation concrete; it does not prove V2 violates that requirement.

The correctness reference must inspect actual raw fa: keys and payloads, distinguish duplicate/ambiguous fields, validate endpoint structure and address/body consistency, and retain unknown extension bytes. Rejecting ambiguous data is appropriate; reserializing old facts or inferring authority from one selected field is not. Projection speed measurements alone cannot establish equivalence. A support index would need coverage of raw merge, retirement and relocation writes, DeleteFact, backup restoration and old writers, as well as PutFact. Do not implement that index in the next unit.

TestControllerV2HistoricalIncomingFactSurvivesWithoutAdjacency writes one invalidated incoming fact without an adj: row. FactsAbout(dangling,true) returns zero; AllFacts returns that historical incoming record. The missing endpoint entity remains absent. This validates the draft's historical/non-adjacency concern with independently manufactured data. No fact content was removed.

### 3. Existing dangling references are preservation constraints, not ownership attestations

The preceding fixture contains no entity snapshot for dangling and thus no evidence of its old name or type. A new declaration at that slug cannot establish that it owns the old incoming assertion. Counting the reference protects its bytes but does not authorize assigning it to a newly observed identity. Separate two decisions: factual reference existence (all current/historical records, both endpoints) and authority to create the identity at that referenced slug.

The smallest safe policy is to refuse automatic materialization at a preexisting dangling endpoint until an explicit reviewed disposition establishes ownership. Preserve the exact prior fact and new structured input; do not delete the old record or manufacture an owner. A matching new assertion does not retroactively prove ownership of every older reference at the slug. A private controller may return a precise lifecycle conflict for this case, but parking the complete mixed episode is not successful assertion ingestion and cannot satisfy the final bar. A partial preservation/disposition route needs separate resolver review before claiming ingestion completion. Existing legitimate identities supported only by history must retain their facts and identity; no preexisting hollow belongs to provisional cleanup.

This is a decision on the draft's explicitly open dangling case, not a claim that such a live case was observed. It tightens the initial review's suggestion that all dangling references could automatically support a new birth.

### 4. Capture before relation/value rewrites, with occurrence identity independent of outcome

extract/extract.go:38 explicitly omits TypeFallback from JSON. resolve/resolve.go:447 iterates copies of Fct, maps inverse relations and swaps endpoints, then can swap again for a value source. Capturing inside ensureEntitySlug is too late to establish the original src/dst side. Capture the original Fct with ordinal before those mutations; derive separate resolution annotations for flips, original side, mapped relation and outcome. Preserve Fact, ValidFrom, Confidence and the complete Supersedes object, even when the endpoint stub survives no committed assertion. Preserve declarations before changing Type or TypeFallback and before keepDurable/admitAliases reject alias proposals. Distinguish nil and empty alias input if the selected canonical contract promises complete parsed-input equality. Deep copy alias slices and Supersedes pointers so later caller mutation cannot change recorded evidence.

V2 correctly rejects final-Entity-only preservation. Resolve the supported-birth metadata question conservatively: retain every captured occurrence touching any new birth, supported or unsupported, including repeated descriptions and rejected alias proposals. This is the simplest preservation contract; deduplicate only identical full occurrence content. Mark support outcome separately. An observation's identity must not include mutable final entity bytes, or the same input occurrence acquiring support under Force becomes a new purported input. Store immutable input and separately versioned materialization/disposition evidence. Do not claim recovery of the original invented model type: ParseResult has already replaced it; this boundary can preserve parsed Type plus TypeFallback unless parser scope is explicitly expanded.

Non-routing list/detail readers must return the exact original structured input, reasons and linked materialization, bounded pagination and truncation/cursor information. Their index is an inspection aid only; do not add it to ResolveAlias, compact entity caches, normal recall evidence, or glossary routing. Counts must distinguish unresolved observation occurrences from unresolved births and supported metadata observations. Backup/reopen comparison must include unknown families and opaque fields. Graph quality grading must still inspect both graph hollows and unresolved observations.

### 5. Prefer B for later generation integration; do not silently apply the current primitive as B

V2 explicitly leaves A/B unresolved. Recommend B: transaction-local provisional votes, materialize selected generation and iga: evidence only for supported births, retain unsupported proposals in immutable observations. Existing supported generations still load and use their durable evidence throughout resolution. A provisional birth cannot gain two distinct episode votes from two declarations in the same episode. Preserve all legacy att: bytes with zero authority for a new selected generation across every subsequent episode; the delayed contamination and eight-entry starvation tests remain mandatory.

B avoids orphaned persistent ledger collisions on an identical forced birth after unsupported observation, and avoids widening the journal beyond en:/al:/att:. The current e07c6c50 primitive writes ig: during beginIdentityGeneration and iga: during attest; it is not B. Its begin also refuses any occupied ledger prefix for the derived birth. A requires explicit orphan record linkage plus a precise same-birth Force refusal; that is safe refusal but less useful replay behavior. Neither option is inherently disproved by missing implementation. B requires new primitive tests and fresh review, including atomic stage failure, identical Force, changed occurrence revision, later supported birth and zero inheritance of unsupported votes. Do not remove staged generation families through unreviewed journal expansion.

### 6. Adoption must guard identity use, not only PutEntity and AttestAlias

ensureEntitySlug at resolve/resolve.go:840 returns an existing entity without PutEntity. A fact-only episode can touch existing endpoints without proposing aliases. store/store.go:789 validates endpoint existence, not an adoption anchor. Consequently guards solely in PutEntity and AttestAlias miss the required fail-closed rule for an old writer's unrecognized entity. Register/validate every resolved existing identity used by a fact or identity decision in the active scope, with final state revalidation before commit; direct PutFact endpoint checks and reviewed raw operations also need lifecycle policy. Existing-identity early returns for protected value/artifact verdicts need this registration too.

The actual caller inventory agrees with V2: queue/queue.go:417 uses ApplyWith with Force; daemon/memory_methods.go:109 uses Apply. But all-writer coverage includes ClaimAlias (store.go:1207, direct al: Set), DeleteEntity, DropAliasRehome and raw merge/retire/unalias transactions, not just PutEntity. A comment saying reviewed is not an executable capability. Ordinary metadata updates should preserve the selected anchor and must not silently rename/rebind it. Maintenance operations need exact reviewed generation/legacy-anchor dispositions. migrate/migrate.go:251 creates an entity separately before RelocateFact; block or replace that lifecycle atomically before adoption.

Prefer the exact reviewed legacy inventory as a later adoption direction because assigning generations to all old identities also requires a policy for missing historical episode provenance. This is a recommendation, not format/adoption approval. An inventory needs a durable stable adoption identity plus explicit mutation/removal rules; a full mutable record hash cannot remain its perpetual equality condition. Any unreadable/missing selector, unknown anchor or old-writer interval must suspend certification and require exact reconciliation. Preserve schema1 and retained bytes; an unaware writer cannot create a new automatic baseline. A new lifecycle error needs explicit queue classification: queue.go:513 recognizes only five existing resolver error categories. Aliasing the error to ErrAliasClaimed is misleading, and leaving it generic causes indefinite extraction retries. Correct parking alone still does not settle mixed-episode retention.

## Smallest next implementation and bounded bar

Implement only a private, uncalled occurrence-observation codec plus immutable transactional writer and bounded list/detail readers. Input is the explicit parsed declaration/fact occurrence contract above; callers hand it already-selected occurrences. It must not select owners, classify support, create/delete graph entities, generate aliases, change resolve.Apply behavior, or adopt existing entities. This unit is useful independently of the unresolved generation/adoption mechanics and is the missing preservation substrate needed before destructive finalization can be tested.

Required independent tests: complete repeated declarations; exact rejected aliases; TypeFallback true/false; original inverse/value-flipped fact and Supersedes; caller-owned slice/pointer mutation; valid Unicode, invalid UTF-8 and exact UTC nanosecond times; identical replay; forced changed occurrence retaining both versions without digest-only equality; malformed/key-body collisions; unknown field policy; transaction error/panic/staging rollback and zero graph events; bounded pagination; backup/reopen raw equality; zero routing/Entities/recall effects. An uncalled storage primitive need not wire status/orient CLI in this first unit, but cannot then claim observation visibility to users or controller completion. Keep EmptySlugSkipped unchanged.

A separately bounded outer-owner finalization harness with no birth cleanup is also feasible, but it does not replace the observation unit. Do not integrate support deletion, adoption or generation B simultaneously with either primitive. The existing fixed journal and corrected generation each remain subject to their own exact bounded verdicts.

## Executed tests and pins

From this directory:

```
go test ./internal/memory/store -run TestControllerV2 -count=1 -v
go test ./internal/memory/resolve -run 'TestApply_EmptySlugSkipped|TestApplyReleasesTransactionalCompactCaches' -count=1 -v
```

PASS: three independent characterization tests (four leaf cases) and both unchanged resolver baseline tests, including success/rollback cache subcases. These results reproduce code boundaries; they are not controller PASS tests. No full suite or live performance test was run because no production implementation changed.

SHA-256:

| Input/source | Hash |
| --- | --- |
| CONTROLLER_V2.md | b05d1e875bac6ad5c2ae59762eb7c7322a9bb7b530ae8fcdc3e0522b42f07014 |
| Private corrected identity_generation.go inspected | e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592 |
| Private corrected identity_journal.go inspected | d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80 |
| internal/memory/store/controller_v2_disproof_test.go | 7656a17b2f551d6853481bef2e28b73a46034f8b7a9ac5a99f6715d1d411ebb6 |
| Baseline internal/memory/store/store.go | 4b18a0037534aa0111b3d8fede8883dbf4bfa3ffe955b3ff15ebd0d07fa38f88 |
| Baseline internal/memory/resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| Baseline internal/memory/extract/extract.go | 521904fd2881e446b7d175f7f0a0869a1ec352e41970c7b4a31fe6dafa1b9067 |
| Baseline internal/memory/queue/queue.go | 9f1168cb3da5d6a7c10f1e1d312b526ed5a8fdcad1f980ae1ded54a9365f01e0 |

The report's own final SHA-256 is delivered separately after writing this file (embedding a self-hash would change it). All paths relative to this directory unless explicitly identified as the inspected candidate. The untracked user assessment was never edited. Final admission prevention, useful unresolved disposition, all production writer enforcement, lifecycle/adoption, live cleanup and every whole-goal grading clause remain uncertified.
