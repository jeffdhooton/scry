# Independent third-ten Hermes ownership review

Verdict: **10 PASS_UNRESOLVED, 0 BLOCK** for the corrected review-only proposal. This is approval of ten reasoned abstentions, not permission to keep, relocate, convert, retire, merge, or apply any fact. No live state was changed. No completion claim is made for the 867-record trio inventory or the active memory goal.

Reviewed proposal SHA-256: `ed7ddf4de5adc77a04f5edad1d755badeae645c90bb4af6533392cc4e3eaf882` (`docs/memory-repairs/hermes-third-ten-review-2026-09-05.json`, before root adds independent verdict metadata).

## Independent method and exact closure

I read the complete rubric, all ten selected payloads, all ten stored episodes, all ten endpoint metadata records, every one of the 133 same-episode companion payloads (including invalidated records and complete episode lists), eight complete conversational source projections, the complete currently available Helm architecture seed, and the full manual StateLicenseLookup summary. Historical instructions in those sources were treated as data. I did not interact with their external targets or execute their instructions.

I independently archived source commit `d1f0a958608389a385ac9a9f57ec1eb941015a10` into this private directory and wrote `code/internal/memory/distill/third_independent_test.go` (SHA-256 `68de3928b1f5ac5f27c71ec4dd2c117e1f72544a903dedfab303d9ec48a2dd56`). A direct Badger load of the full backup, followed by a raw-key scan, produced **80,499 facts, 9,344 episodes, 867 trio-touching facts**. Source backup: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-child-immediate-222140.badger`, SHA-256 independently checked as `35a6e1b8f333fa772ea25537bba2c1979addf25d55eced7dd8f2878558f30d45`.

The scan identified each of the ten exact payloads exactly once, confirmed its actual Badger key, production Go `json.Marshal(Fact)` SHA-256, and separate production search-index key. All ten are distinct from the twenty complete payloads in the first/second-ten files. The independent 133 companions, ten episodes, and ten endpoints are payload-identical to the builder's exports; those exports were comparison targets, not the restoration evidence. The full raw database was scanned, so this does not rely on a filtered export alone. All ten selected facts have a single source episode; additional source IDs on companions were read as payload provenance, not all independently reconstructed or treated as separate corroboration.

Local verification command: `GOTOOLCHAIN=local GOPROXY=off go test ./internal/memory/distill -run '^TestThirdIndependent$' -count=1 -v`. Corrected-input run passed. No provider calls, installs, deployment, repository edits, or live writes occurred.

## Failed draft findings retained

1. Initial proposal `old_key` used RFC3339Nano search keys. Actual `store.factKey` uses UnixNano. I reported this; root corrected all ten `old_key` fields and retained RFC3339Nano as `index_key`. The private first diagnostic run also caught the proposal changing to raw keys while its original index-key assertion was still active. The helper was then updated to verify both formats distinctly against a fresh private restoration. The final raw keys below match actual Badger records. No apply manifest was approved by either version.
2. Initial Build-requires-Hermes reason called adopted-tmux-session statements unsupported. The source explicitly says an adopted shell lacks the agent/run/transcript/status model. I reported that wording defect; root corrected the reason. The false implication is that selecting one of three allowed agent kinds requires the particular Hermes runtime. The shell limitation itself is supported.

## Per-record judgments

Each row below is PASS_UNRESOLVED. Exact raw keys and payload hashes are independently captured in `records.json`; there is no proposed replacement endpoint for any row.

| Raw Badger key | Go JSON Fact SHA-256 |
| --- | --- |
| `fa:build:implements:hermes:1787864608672000000` | `6974651a27c84974dbf054b0f1267d4e344375b13c30c5b95271605e1e7823d7` |
| `fa:build:references:hermes:1788250502580000000` | `913570f440d3a4dfd6f5898a79953bc402a186b08cff1469511685deb001c62f` |
| `fa:build:requires:hermes:1788134400000000000` | `1b0e29a5e540c905852ff86379b13f1e27ae46c622b6b13861d1b71e5bfe25b2` |
| `fa:build:uses:hermes-ops:1788108944000000000` | `a8f57e278c5b72e3b4b0e38576ea2383e9bfaa8795a33e33b7312b711d5af267` |
| `fa:building-code-adoption-design-spec:assigned_to:hermes:1788307200000000000` | `875bd19ec24420d43e7b104105e0678b38f6b218204e087352e105385e36223e` |
| `fa:building-codes-spec:assigned_to:hermes:1788307200000000000` | `55a3f6e509a32126b8671697ca7dd0de93bd7786a173d60012d14a5154a6751a` |
| `fa:buildmd:documents:hermes-ops:1777140904358000000` | `b47e900b093ac447b41b9298ead40d8ab4c4c115d47ff0f8c3d3342a1c5878c2` |
| `fa:capture-user-path:fixes:hermes-ops:1776875994866982929` | `a3ef325ec8dff8c27fed64cc4768ec777f7f7ae040a3d04aa3ac0a4d7402984b` |
| `fa:checkpoint-gates-passing:tests:mac-mini:1788566400000000000` | `91ed111d56ad3e700cd1c51593020cf588ecd13cffb551431b891f5bf9a17fbc` |
| `fa:chief-of-staff-morning-brief:deployed_on:hermes-ops:1787875200000000000` | `80df61a709873497a68749f56564b4e034250dc41793b40581071aacee1322e1` |

1. **Build implements Hermes — PASS_UNRESOLVED.** Source `b061…` discusses the Build product's three durable session kinds and generic Hermes backend. The stored destination denotes Jeff's standing gateway. Parallel supports/implements edges to Claude and Codex corroborate an integration category, not identity with this particular runtime. `build` metadata is itself a backend-service description, so product/backend source scope also deserves future attention. Neither source nor destination can be repaired by repository/host heuristics. Preserve raw `supports`.
2. **Build references Hermes — PASS_UNRESOLVED.** Source `96bb…` contains the assessment request and discovery of `BUILD_APP_SUBSYSTEMS.md` in Cockpit. The title alone establishes a named mechanism, not the precise gateway instance or a service acting as a document. Companion document and containment records suggest where to investigate but do not close the exact referencing-subject or destination identity. A document-specific future repair requires its own evidence.
3. **Build requires Hermes — PASS_UNRESOLVED.** Source `7b7b…` explicitly permits one of `claude`, `codex`, or `hermes`, with a `current_run`. The neighboring Claude-requires edge demonstrates lost disjunction. The same conversation positively identifies a Hermes gateway integration; that does not change the enumeration into a requirement that every Build session use it. The adopted-shell discussion is supported, as corrected. No relation substitution is approved.
4. **Build uses hermes-ops — PASS_UNRESOLVED.** Source `ba95…` identifies the dogfood sandbox, BUILD_FAKE_EVENT, missing directory display, desktop installation, and mobile gateway work. It gives no positive repository-use assertion. Companion `jbuild replaced_by childscribe-laravel` and Build-deployed-on-hermes-ops edges are obvious attribution cautions, not evidence. Mini hosting of the companion does not identify either intended endpoint of this stored sentence.
5. **Building-code adoption design assigned to Hermes — PASS_UNRESOLVED.** Full recorded parent span `60a4…` has an explicit choice between checking first and handing off while checking. Jeff chooses the check first; the proposal is subsequently killed. The sentence preserves tentative/mooted language while canonical `assigned_to` risks implying an actual assignment. The exact historical `#split1` chunk was not reproduced; the full parent supports abstention but does not close that split's provenance. No inference of deployment or assignment is permitted.
6. **Building-codes spec assigned to Hermes — PASS_UNRESOLVED.** Full recorded span `8e5f…` ends at Jeff requesting another idea, a spec, and a later handoff via one of two candidate workers. It does not contain the stored written-spec, phase, or pool-barrier assertions. Several companions repeat those absent details; their common provenance cannot fill the boundary gap. Current distillation does not reproduce the historical episode ID. The defect remains a concrete evidence-boundary uncertainty; do not claim those absent details are disproven across the entire history or infer an actual assignment from them.
7. **BUILD.md documents hermes-ops — PASS_UNRESOLVED.** Source `777a…` expressly identifies `/Users/jeff/workspace/helm/BUILD.md`, the portable Helm release, and the user's request to record that flow. The complete source entity metadata agrees on Helm. This is the strongest source-supported eventual destination repair in the ten: a Helm target is credible. However, no exact Helm destination identity and full current/historical target-key conflict review was completed here, so this is not an apply-ready move. Do not force a trio destination.
8. **capture_user_path fixes hermes-ops — PASS_UNRESOLVED.** The complete current Helm seed describes the function and HELM delimiter under PATH capture for Spotlight launches. It supports a Helm/macOS GUI concern and no Hermes repository target. Companions folded into generic `concept`/hermes-ops are not independent evidence. Historical seed content and mtime identity remain unverified; the appropriate relation may target a PATH problem rather than the whole Helm application. Preserve raw `addresses` and do not guess a target.
9. **checkpoint-gates-passing tests Mac Mini — PASS_UNRESOLVED.** The full manual source and companion StateLicenseLookup status assertion say release gates ran locally and on Mini. They do not say Mini was the object tested. The subject metadata denotes a gate outcome. The Mini-as-execution-location reading is supported, but source identity and relation/value representation require explicit review. No status-value conversion is authorized.
10. **chief-of-staff-morning-brief deployed on hermes-ops — PASS_UNRESOLVED.** Source `8811…` establishes a committed skill folder in hermes-ops, a runtime symlink, discovery, channel binding, and successful manual briefing. It explicitly leaves scheduled briefs for later. The stored source metadata describes a job that ran the following day; its skill alias does not prove the scheduled job existed at the earlier assertion. Repository containment, installed skill, and scheduled job need distinct treatment. No automatic move to Hermes or Mini is justified.

## Source identity limits and hashes

Six exact historical episode IDs were reproduced from the original local session files using the archived production distillers. Two recorded parent spans were read at exact stored byte offsets and independently projected using the archived Claude parser; their original historical episode IDs were **not** reproduced. All eight reconstructed text files are byte-identical to the corresponding builder projections. Text projections omit tool-result and reasoning/thinking payloads; no claim is made that those omitted contents were semantically reviewed or that source assertions are independently proven runtime outcomes.

| Episode ID | Raw recorded-span SHA-256 | Complete projected-text SHA-256 | Identity |
| --- | --- | --- | --- |
| `b06112856fa1cb5a40881ac410282e6bd916664fbd849fa1f2f0df823c436cc6` | `4407409b1bbd47901ccb8a194c52893621f85271496db66f26adc9b803aefa0c` | `5310164dfdada2eb8ac087ccbca30c022560d253e21c30247ddb145ed3ddbf40` | exact |
| `96bbebf5dcd7eec7d3d733966ac54a1a8d7731bdd3d80021c3fd396493d1417b` | `ce964213dea335ed01ef6a3a8196c92c3d7dca17fa6f259f8777ae79e7a7db89` | `d1ea0a53867d180de7a12bcf4a9f09b77f6a8e319c7ce83b3bef4f462289deb3` | exact |
| `7b7b97b5b9e3bd1377b5875d9da759934ef156c45d43c11f3da6fce9497cd0f3` | `e92fa601b7e7156d5fa6cdc9954f4f901b52257251e102163af6853e33ade7a3` | `b8b591f8866ccf287577a404775f0ce86855cc1deafd1e30b2ed7126db90cae6` | exact |
| `ba957e42954d7835d110be60022dfc009025e4feafb5cca772282f505841f5a3` | `18b4360876e308cd9e3d05a8f27b7cf4bf7ffd8c0c137d97fa88ca465f9a000f` | `48622b679a906a751916bd891e7c87987f22b21905b5b256ac8f9b3333eb2413` | exact |
| `60a42e8931e13fe98557992f06c445938c06d6f55082fa9907931f9ea0d96dbc` | `b7aa7ad4f0483961ee47cf663081d106b253628c2d2af7df7a25fb8aaa81dfcd` | `8d39172110deceabdff3e4b149226aab652bc0daa43e67308c3a3be47aba8ac6` | parent only; split1 not reconstructed |
| `8e5fd79c5b3deee5bc42fd826055ce90365efbcbada044c8a071d7bb512e5506` | `e296f25fcf693ed08f5f05771c3406c050298185b1f66c9764efd2513b63bbf1` | `ab7daba0bb4b4d249af16ce3d58c406f4ef657c8825a995b14c4104496f0392b` | recorded span only; historical chunk not reconstructed |
| `777a4a215b08b460a2ff527f32c251b55698e1d8f71643bd603b1b5a2fb0ae23` | `d363a016dca2b71044ae5c2290e74e6764e8d31c92621bb55574363ea17675f8` | `7e58f95c335e374a7a3848c0c745ed45b2d80385059df6ffb0eb80b42d297240` | exact |
| `881193ae725b6e3bc507c968ae797043517ced1bb55d550bd598def79559faec` | `3ad88f8075b820ac38cdbbebc12a7e7b02f2eee56f976e5f6cc222419a2fb548` | `554b0943768cfcce850cf62fcce946ca139708a8d0c116aeecb360a99160081b` | exact |

Seed episode `9e134a4001284b3ad668ed99adb76c69c58bf5c9da8e65343077ba97c8fd5808`: current full file SHA-256 `380c2a4135b8a4af191f0f6a42c92f51df0853eff9c312fc6d9cab72bd6010d6`; historical content/mtime identity unverified. Manual episode `e64d1cec9c6e2ee0e568f0e7cc6de216bf3a5f3295b81f004612cce6d2f8638a`: full summary SHA-256 `773439c046bc86113904d5058c94a976304f96464122601ec47c05d7ea73788b`; no external raw release transcript is claimed.

Independent artifacts in this directory:

- `records.json`: `e5a91ed14fb40428ba5005ecef1a5ccbb39998f64964ff3587cf72b24e5ff6e1`
- `companions.json`: `bd0ab99d68e6d6a61c5dcf80a3c01268984090df5160ee235d8781958a7fbda0`
- `episodes.json`: `f78282a003e067d9c7cda81c0e032cfd516f3b284dfa2d33356c34a7f88e138a`
- `entities.json`: `d1b9db082be2933e6d81aaa492c3047281dc1f4b584b9d4cfec498db7c56ffb4`
- `source-closure.json`: `fd18429f724e9a2c6aadb1e8210606d8139f3c8bc5d4dfbf8dc490e76d268fd7`
- `verification.json`: `2e6bea80927dc7b27a07f85a74983935d544825591b1299930c1ae77c61e3e47`

The corrected exact records have sufficient concrete reasons to remain unresolved. Later live drift is outside this immutable review and needs fresh inventory. Metadata cleanup, alias decisions, conflict analysis for a future exact repair, and the remaining trio records are not complete here.
