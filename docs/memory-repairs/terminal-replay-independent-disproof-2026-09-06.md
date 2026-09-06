# Terminal replay — independent bounded semantic disproof

2026-09-06. **NO-GO for the frozen terminal replay implementation.** One independent semantic violation is reproduced through actual `prepareCompleteAdmission`: previously accepted actors that were already generation-backed at original preparation can silently bind a different generation at the same stable entity tuple. Four variants fail: primary source, primary destination, exact hint-effect target, and completed declaration. Every call preserves actual raw rows and emits zero events; the defect is acceptance of stale identity authority in a draft terminal result, not a committed graph mutation.

The root reports a separate correction and reproduction of these unchanged tests. This report grades only the immutable frozen code and does not independently approve that correction, delivery, production activation, or the full goal.

## Scope and provenance

Review input `/tmp/scry-terminal-replay-review-sep06.khrGn1/REVIEW_INPUT.md`, SHA-256 `518d34ea7546d512e9276cb581f99f537758d9d13277251668e32782db10e035`. Freeze manifest `/tmp/scry-terminal-replay-review-sep06.khrGn1/TERMINAL_REPLAY_FREEZE.json`, SHA-256 `8dcad6f02e5cdf0e84da4c8129b3e917f5c143aec39ef529db607fe472af6e16`.

Private export `/tmp/scry-terminal-replay-disproof.Re0hjU/code` is a fresh `git archive e097fa6`. All 83 additions were copied using apply_patch after full hash/size checks. The 600 baseline SHA-256/git-blob pins and all 83 addition SHA-256/size pins match both exports at start and end. No supplied, frozen, shared, or root source/test was changed. The copying/verification script is `pins.rb`; logs are `initial-pins.log` and `final-pins.log`.

I ran and read required session orientation first; read the complete active objective, controlling assertion contract including independent addendum, semantic bridge experiment/disproof, ControllerV3/assertion-identity/identity-mutation reviews, ordered-overlay contract and incorporated disproof, prior complete-preparation report and root correction/progress evidence. All 24 admission source/test files were read completely. The actual overlay declaration/primary/hint/lookup/alias/control/query chain, registry, owner/coordinator, raw FA decoding, identity/FA inventories and ledger finalization, input/observation/EP/head binding, and relevant legacy/generation recognition methods were inspected. Birth-registration and episode-selection contracts were also read for their narrower history/ordering boundaries.

All new semantic preparations call actual `prepareCompleteAdmission` through the supplied `admissionReplayDraft` helper, with only canonical original input and actual supported EP bytes. Prior selections and graph changes are explicitly synthetic actual-row fixture setup, necessary because the production materializer is absent. No fake owner, alternate support set, replacement witnesses/finalizer, or alternate preparation algorithm is supplied. Full raw equality and zero events are checked around every wrapper call, success and error. The actual wrapper also rejects nonempty returned identity or fact writer histories; its fixed registry verifies complete final inventories before success. The retained suites cover owner, ledger, and failure boundaries. No separate externally observable error-history API was invented.

No actual graph stores, backups, replicas, SSH, providers, credentials, config, hooks, rooms, remember, deployments, sweeps, or extra agents were used. The mandatory initial orientation was the only Scry interaction. Authored source/scripts/report used apply_patch; gofmt touched only new tests. All Go commands used `CGO_ENABLED=0` and private `TMPDIR=/tmp/scry-terminal-replay-disproof.Re0hjU/test-tmp`.

## Proven violation: existing generation identity is not pinned on carry

Reproducer: `code/internal/memory/store/terminal_replay_fresh_generation_test.go`, `TestFreshReplayExistingGenerationCannotChangeBehindStableEntity`. Source SHA-256 `8bab674931fd5c7faa85eea81f876e1089739d4d81e5e1519fc85251c1cad2e4`. First log `generation-disproof-first.log`, SHA-256 `2687e65d8b8219f7cc1d08b4236bc9eacae7eb3655110a8bb049443dae17ce0e`.

The fixture creates canonical EN/IG/EP rows for Atlas, Borealis, and Cygnus. Their original generation birth is under `original-birth-proof`, occurrence 0. Actual complete preparation accepts an assertion or declaration using those existing generations; it creates no birth receipts. The fixture seeds the prospective exact FA/effect/identity/result/input/head rows as a synthetic prior selected result.

It then changes just the targeted actual IG to another canonical generation, under `replacement-birth-proof`, occurrence 1, and adds that valid EP proof at the same birth instant. The actor's EN name, slug and creation time remain identical. All original FAs, original EP proof, original result/input/head and original selector witness in the receipt remain intact. No explicit repair lineage connects the two generations. The new actual selector is structurally valid but denotes a different complete generation identity.

Actual terminal replay returns nil error and byte-identical prior result in all four affected roles:

| Actor role | Old IG raw SHA-256 | New IG raw SHA-256 |
| --- | --- | --- |
| Primary source / completed declaration, Atlas | `4dc766efaefedbe13135a655c0000e9d58291653bde12b96434adccd0983c30c` | `0de20c7c7de15220381b2e97a9727f9bfe657ca7249044a3d4a704a69b3536c1` |
| Primary destination, Borealis | `40450d43b2e3b990e26ae2a1724207fe7eadad9c211f958c48c38a0d1c2f1f01` | `12d2c4e79648e2e38eac6cf1e2ab92b6e685e1f38f8b4625bcade522c6d26132` |
| Exact hint-effect target, Cygnus | `802043e8f2553ea9b41c2a2724d80d1218006c9309afdd5046dbc8f5e41b046a` | `3ee89c84f892f2b53d4979fba7615a8988973bfd5fd51ab706a20fe388833863` |

The same test's unchanged-generation and later-description-only controls pass. This distinguishes stable mutable metadata from generation identity and avoids requiring whole EN byte equality for legitimate descriptive changes. The four failing assertions are unchanged from their first execution and fail again in the final full repository run.

Cause: `identity_assertion_admission_replay.go` builds `expected` entities from prior receipts, then checks actual Name/Slug/CreatedAt and recognizes current lifecycle. It compares the exact prior generation only when the slug occurs in `previous.Result.Births`. An actor already generation-backed before this episode has no such birth receipt; its pinned original `ig:` witness is never compared with actual current IG. Current recognition establishes that some valid generation exists, not that it is the previously accepted generation. The affected closed hint target is also accepted because exact FA content and interval equality cannot establish endpoint generation identity.

Violated requirements: the review input explicitly requires stable endpoint identity/lifecycle/generation proof and challenges recycled slugs/birth generations. The ordered-overlay contract says a recognized generation requires its complete canonical IG identity and original EP proof. The controlling assertion contract requires carried accepted work to preserve its exact dependencies; changed identities need explicit recorded repair lineage or stale/conflicting evidence, never a matching textual endpoint. This is a bug in the supported terminal path, independent of the missing materializer or partial-declaration mode.

Smallest correction: for every accepted preexisting actor, bind the original selector/lifecycle witness, including relevant absences, to actual current selector state before terminal suppression. Require consistency when multiple carried receipts reference the same actor. Keep existing mutable metadata allowances and the distinct exact-generation check for births created by the selected input. A changed generation without explicit approved lineage must refuse; it cannot be replaced merely because EN's stable tuple matches. Do not rerun the terminal occurrence as a workaround.

Reproduce from the private `code/` directory:

```sh
TMPDIR=/tmp/scry-terminal-replay-disproof.Re0hjU/test-tmp CGO_ENABLED=0 go test ./internal/memory/store -run '^TestFreshReplayExistingGenerationCannotChangeBehindStableEntity$' -count=1 -v
```

## Other fresh challenges and preserved fixture failure

`terminal_replay_fresh_semantics_test.go` contains six additional top-level groups, all passing after one fixture setup correction:

- Terminal nil, present-empty, no-target, and accepted inverse hints keep byte-exact original receipts and endpoint roles after two additional eligible/ambiguous FA targets arrive. A separate deferred input does not cause hint execution or target acquisition.
- A terminal two-value non-assertion remains terminal after actual recognized numeric identities arrive; its discovery is not repeated.
- A supported birth first discovered by a rejected primary retains its original discovery and accepted support receipt on retry.
- An unsupported discovery gains independent retained support when its deferred hint identity becomes recognized; the rejected discovering primary stays deferred and the complete first birth/observation remains original.
- Missing/corrupt actual input, head, result, EP, FA, changed input revision, rewritten closed end, and missing exact FA address each refuse through the actual wrapper with full raw equality and zero events.
- A completed declaration whose accepted alias is later removed or assigned to another recognized actor keeps its historical declaration receipt, defers its dependent consumer, and still accepts independently useful deferred work once that work's hint route becomes available. It does not refill cleared metadata or reacquire the alias.

The initial supported-birth seeder called the supplied helper, which uses PutFact, before creating the synthetic supported EN rows. PutFact correctly refused missing source `draco`; that leaf never reached replay and is not an implementation finding. Complete original source `fresh-semantics-initial.go.txt` (SHA-256 `b4743dc06dbbb5273c5eee75d0c255851a697990fbd2d84a38b0fda88f8597ac`) and `fresh-semantics-first.log` (SHA-256 `cf29def107d8d314702cdeabd1624f3db09d245a37ecfff029568ecb689679e6`) remain preserved. The new helper was corrected to seed supported EN/IG first; all original semantic assertions were retained. Corrected log `fresh-semantics-corrected.log`; the later additive declaration case is `declaration-binding.log`. No supplied test was changed. Read-only path guesses for two overlay filenames and private delayed-design documents failed before locating the actual source/contracts; no data was changed by those reads.

The original complete-preparation collision correction and input-confidence/address-evidence regressions pass unchanged in the supplied full suite and final full suite. Retained terminal tests additionally cover exact accepted FA/provenance/closed-end proofs, later valid closure with unchanged result lineage, cleared metadata/removed aliases, actual retained vote rows versus mere EP presence, and deliberate partial-declaration refusal. The generation controls and supplied owned-receipt tests remain intact.

Root separately reported unfinished structural/evidence-validator omissions during this review. Those leads were not copied into this export or presented as independent findings here. The generation finding was independently derived from the replay source and reproduced before root supplied its correction. No missing future writer, full evidence validator, granular partial-action replay, or changed-revision writer is graded as an implemented terminal-semantic bug.

## Executed checks

Complete stdout/stderr is retained in the named logs through tee under pipefail. Exit statuses are recorded in `CHECK_STATUS.json`. Durations below are command outcomes, not p95 or comparative performance measurements; some checks ran concurrently.

| Check | Result |
| --- | --- |
| Initial/final 600 baseline and 83 addition pins, both exports | PASS |
| Supplied `go test ./... -count=1`, before authored tests | PASS all packages; store 74.976s, resolve 18.648s; `full-supplied.log` |
| Fresh generation disproof | FAIL four semantic leaves; two controls pass; package 0.799s; `generation-disproof-first.log` |
| Initial five fresh semantic groups | FAIL one fixture seeding-order leaf; other executed groups pass; `fresh-semantics-first.log` |
| Corrected first five fresh semantic groups | PASS package 0.789s; `fresh-semantics-corrected.log` |
| Additive declaration binding/current independent utility group | PASS two leaves, package 0.347s; `declaration-binding.log` |
| Retained Admission/Overlay/IdentityOverlay/DelayedBirth/OrderedContract/Policy selection | PASS store 40.604s, resolve 2.195s; `retained-tests.log`; identitypolicy regex selects no tests but full suites cover it |
| Final `go test ./... -count=1`, all supplied and authored tests | FAIL only the four generation disproof leaves; store 112.056s; every other package passes; `full-with-disproof.log` |
| Relevant store/resolve/identitypolicy vet | PASS; `vet.log` |
| Exact pinned policy-token verifier, corrected existing manifest | PASS 157 declarations / 248 symbols; `policy-tokens.log` |

## Verdict limits

Bounded NO-GO applies only to the frozen supported terminal replay program. Preparation labels are draft intended dispositions. These fixtures do not prove a production commit or malicious arbitrary raw-writer resistance. The exported program remained read-only in every exercised case.

The root's separate correction, future complete evidence/action/lifecycle validator, explicit repair-lineage replay, immutable result/materialization/vote writer and final actual inventories, normal Apply integration, policy deduplication, all-writer/schema floor, adoption/deployment, actual backup/rollback, cleanup, recall/benchmarks, sweeps, and two full-goal grading rounds are not approved. Partial declarations, changed revisions and V1 acceptance-unknown deliberately refuse within the stated temporary boundary. This report neither changes that boundary nor grants delivery or full-goal approval.

All authored test/script/report/log hashes and external input pins are indexed in `EVIDENCE_SHA256.txt`; this report and index hashes are sent separately after writing. Root remains sole shared/live writer and posts the verdict.
