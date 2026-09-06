# Birth registration — independent bounded design disproof

2026-09-06. Conditional GO for a private, uncalled registration wrapper after fixing the ordering, freeze coverage, and typed-result contracts below. The architectural direction is coherent. The frozen design's baseline/current absence checks alone do not establish its promised before-first-mutation ordering. No registry implementation exists, so this is not a registration code PASS, controller approval, or live-goal verdict.

I ran and read session orientation first, read the complete active goal objective and frozen birth design, original ControllerV3, its complete archived independent review, complete identity mutation/current episode reviews, and complete serialized coordinator code review. I inspected canonical input/observation, generation, relationship inventory, identity/fact ledger, owner and relevant resolver sources. I exported baseline `4f1d1beb758a8c381a8b5184b790db8dd651ca6c` to `/tmp/scry-birth-design-disproof.uWc8AE`, copied only the three pinned reviewed coordinator source files, and authored only a synthetic characterization file and this report there using apply_patch. No candidate/shared edits, real database/replica access, providers, rooms, remember, sweeps, backups, configuration changes, deployments or agents occurred. Tests use synthetic temporary stores.

## Proven characterization and required ordering correction

`CGO_ENABLED=0 go test ./internal/memory/store -run TestBirthDesign -count=1 -v` passed, exit 0, store 0.525s. The log is retained. These tests characterize existing primitives, not an unimplemented registration method.

`TestBirthDesignAbsenceDoesNotProveBeforeFirstMutation` captures both ledgers before body writes and exercises three cases: `al:atlas -> atlas` remains present; that alias is written then deleted; `en:atlas` is created then deleted. In all cases baseline/current en, ig, il, il-consumed, rs and rt selected keys are absent and baseline facts are empty at the proposed registration point. Both ledgers then verify successfully. The two transient cases have identical baseline/final relationship digests. Actual ordered history nevertheless contains writes for Atlas before this point.

Thus capture-before-body is necessary but does not itself establish registration-before-write. Refusing only unmatched creations at freeze is also insufficient: a late registration can retrospectively match the earlier creation, and an alias-only history has no entity creation to match. This is a concrete ambiguity in the executable checks, not a claim that the draft's stated ordering is undesirable.

Smallest correction: record a registry-owned identity-history position for each first registration. Reject a first registration if this owner has already recorded any en mutation or alias mutation on behalf of that slug, including mutations subsequently undone. Every relevant creation and alias proposal/write must occur after first registration. Baseline raw alias claims remain a distinct, measured preexisting relationship case. Where proposal work is only in memory, the eventual B/resolver integration must gate proposal entry itself; an en/al ledger cannot prove that no earlier memory-only alias proposal occurred. Do not claim that guarantee before that producer is connected. Untracked transient raw writes remain excluded exactly as in the ledger/coordinator reviews; final snapshots cannot detect them.

Choose exact recorded creation coverage for this next unit's freeze. After both ledgers verify, require every recorded absent-to-present en transition for a baseline-absent slug to have the prior same-registry registration and an exact stable birth identity match (slug, canonical name, creation instant). Include entities created then deleted during body. Registered-but-never-materialized entries remain descriptive proposals; do not invent a committed/support disposition for them. Reject deletion/recreation or stable-tuple replacement without a separately specified lifecycle operation. Existing baseline identities are outside the new-birth creation set. This is mechanical coverage of owned recorded history plus verified final inventory, not approval of metadata, aliases, factual support, or arbitrary raw-writer coverage.

An accounting-only implementation could be safe if clearly named and if it explicitly exposes unmatched/late activity, but leaves the central timing question open for another unit. The stricter recorded-history coverage above is the smallest coherent next boundary I recommend. It does not require full resolver support selection or undo policy.

## Exact derivation, duplicates and mention links

Derive from the owned decoded original revision and exact matching primary observation, before relation inversion or value-source flips. The source currently swaps local fact Src/Dst during inverse mapping. A birth originating at original dst remains original dst even if it becomes the resolved source. Preserve a separate future resolution mapping; never relabel the captured observation. Supersedes-only names are never birth origins. A hint link must use a closed explicit supersedes-src/dst role, verify a nonnil original Supersedes field, and retain the exact primary observation carrying that hint. A role cannot reinterpret an absent hint or rewrite first origin.

`TestBirthDesignEndpointTupleDoesNotIdentifyFirstSide` matches both original sides of a same-name fact to the canonical revision. Their observation keys differ, while `identityBirthRecord` produces identical birth bytes because its tuple has endpoint origin/ordinal but no side. That is not a generation-code bug: the design separately stores first observation identity. It means duplicate registration must compare the complete first observation key/raw (and role), not only slug, generation ID, or birth bytes. The other side is an explicit mention, even when its text matches. Same-slug different names or repeated conflicting declarations likewise never replace first birth. Full metadata, alias ordering/duplicates, TypeFallback and unparsed ValidFrom stay in the linked observations.

Check the genuine receiver owner/facade/phase and poison state first, then exact canonical observation validity. For an already registered slug, an exact first-observation retry returns the same handle even after its en record was materialized; do not apply FIRST-registration current-absence checks before this idempotent branch. Different occurrences require mention-link or explicit refusal. Genuine handles must be bound to registry membership as well as owner; a copied structural tuple is not a capability.

Mechanical mention linkage can describe a proposed attribution of an arbitrary matching occurrence to a genuine handle. It does not authorize resolving that spelling to the birth. Keep links descriptively named and prevent their consumption as alias routing, support, B evidence, or approved resolution annotations until the fixed policy verifies them. Otherwise the explicit mention API would simply become a caller-selected ownership override. This unit can preserve competing proposed mappings without settling semantic identity; any future uniqueness constraint belongs in that reviewed policy.

## Typed deferral and failure precedence

Freeze this contract before implementation: return a closed normal disposition (registered, existing/not-new, deferred-preexisting-reference) separately from an error. A valid deferred-reference result has no handle/birth/generation authority, owns the exact observation key/raw and typed endpoint role, and is recorded for the report. It must not call the generic poison helper or return a generic queue retry error. An existing en identity is an explicit not-new result, not silent adoption. Structural misuse, malformed/canonical-mismatch input, foreign handles, wrong phase, prior poison, occupied forbidden lifecycle/retirement keys and storage failure are errors and poison the current scope.

Check phase/owner/prior-poison and canonical validity before any cached idempotent result or deferral. For a proposed absent identity, validate forbidden lifecycle/retirement occupancy before returning a reference deferral: old references must not hide a consumed/retired identity or corrupt selection state. Preserve the owner's first poison at outer return, while local errors remain sanitized. A previously deferred observation never turns into a new birth merely because its baseline historical reference is deleted during the same body. Baseline inventory remains authoritative for the conservative refusal.

Required mixed test: one well-formed dangling-reference observation yields a recorded normal deferral, an unrelated valid registration/write proceeds, and both ledgers verify. Complement with prior-poison plus otherwise-deferred input, malformed matching input plus old references, and occupied consumed/retirement key plus old references; none may downgrade error to partial success. Future complete controller still needs dependency closure before both resolver fact phases and durable visibly partial episode results. A private in-memory deferral report does not complete that ingestion work.

## Replay, persistence and finite scope

The draft correctly refuses to certify identical Force inventory from new transaction births. On identical Force the earlier supported birth is present, so the new registry can correctly produce no new birth while the complete selected episode result must retain the prior birth/outcome. Keep replay/current selection out of this unit. Future replay requires exact prior selected result and revision proof, not remembering a slug or treating an existing en row as a new registration. Changed extraction and consumed identities require the earlier current-episode/lifecycle contracts; omitted occurrences never authorize deleting facts or votes.

The wrapper can purely decode and own canonical input before body because ep is currently written last. It must not call provenance writers early or insert ep before normal Apply's idempotency test. This unit's report is registration accounting only in the provenance sense: final persistence must separately prove the actual ep identity/instant using the reviewed writer. It must not claim durable observations or useful status/orient yet.

Implement one private wrapper owning the revision, serialized owner, both fresh ledgers, registry/handles, normal deferred records, ordered mechanical history checks and one fully owned deterministic freeze. No production callers, caller-supplied baselines/support flags/prior registries, selected generation materialization, automatic adoption, current-result selector, cleanup or resolver routing. Existing private harnesses and their documented counterexamples remain unchanged. End-of-body freeze does not account for later finalizer writes; report snapshots are not postcommit certificates.

After the frozen draft review, the lead proposed a further refinement: the new private wrapper accepts only a registry body callback, fixes its own registry/both-ledger verification internally, and returns its report only after outer commit succeeds. I recommend this smaller fixed boundary instead of a new caller-selected finalizer seam. It makes omission of this unit's accounting impossible, and an outer error/panic must produce no usable report. Keep the old finalizer-injection harness unchanged for its existing evidence. This refinement is not silently attributed to the frozen design and is not tested as implemented. Its report remains a committed transaction's mechanical registration/accounting report, not factual support or full admission success; until the later full policy, a registered unsupported entity can still commit. The reviewed coordinator was committed as `e13f7ae` during this review; the three source hashes below remain exact. The shared full-suite result reported by the lead was not executed or graded by this reviewer.

Independent code review should challenge the three executed pre-registration cases, never-materialized and transient births, unmatched creations, wrong name/creation instant, canonical inverse/value-side capture, hint-only origin refusal, identical retry after materialization, distinct same-tuple sides, repeated conflicting metadata, foreign/copied/closed handles, caller buffer/report mutation, typed deferral precedence, malformed selectors/opaque occupancy, rollback/panic/storage errors, and no graph/evidence/event writes by registry methods. No production prevention, lifecycle adoption, support/undo policy, useful Force result, latency, recall, two-sweep stability or complete goal clause is certified.

## Pins

| Artifact | SHA-256 |
| --- | --- |
| Frozen birth design | 19474eeb08d39c5f9b22ee4cade609bec54bbd29518533321a6ee24f265855ed |
| Original ControllerV3 | c43e0634e1ee14e8177ab505680e75725c7602abfaefcb7b84a2203c7b60d36c |
| Archived ControllerV3 review | 2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10 |
| Identity mutation design review | 94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3 |
| Current episode design review | a9722fcfad3e21980215dfda2e7ef599ae28216bd4a269dceb616dcf2e5845ce |
| Serialized coordinator code review | cc5b969782b03b8fdaf6411f2d691a2e1e8955e6c4d4899e8c0aafaec3cd657f |
| identity_input_revision.go | b69e6daf91e98fba34166fe949c374d65f9196831d49ddcd11b9650364e59908 |
| identity_mutation_ledger.go | e43d2b90840774ea732a8b22299da5fd3b919d8722b2e7f4525d65e8401d2767 |
| identity_fact_ledger.go | 2040e19a3e0bcf505b6faf8253551e687f6d4dc2059823aeeb744536cb8697c7 |
| identity_generation.go | e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592 |
| identity_relationship_inventory.go | eae817f6892b5ec6c2151e4467b187fc9eee55311b1705787e773bbbcc679a2e |
| identity_observation.go | 83a39c03435185963a4a2931f94f30f03a7bd1a86c6e4c545ece95580ad9e0ab |
| resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| store.go | 8093c39a9214e1dbe5546a8cdd5a8cd8c76836952bf2a8616630eeda3ba8f7b1 |
| pending.go | f51c3dec4f7aef2a93e5195d229541ba7253e244606e806fe00d92541df6afb2 |
| identity_admission_owner.go | 396389b06177ce1cd4af18f23412eea4e1f747696d93fc794d82025bfc89dac1 |
| New birth_design_characterization_test.go | 14551cf467d4723cab74dd6a631b4a1744dcde575f3359e6f8287bd8ec118cd8 |
| birth-design-characterization.log | 1823bd2ee1605b5924023718e833eedc54c2431f28c2f115126846b421b5cba1 |

Source paths are relative to the private export's internal/memory/store unless specified. External designs/reviews use the task-supplied paths. This report's hash is supplied separately.
