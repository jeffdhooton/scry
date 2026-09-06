# Independent Child dev-client negative-ownership review — 2026-09-05

Bounded semantic verdict: PASS for proposing owner-specific removal/rejection of exactly `Expo dev client`, `dev-client`, and `dev client` from `childscribe-laravel`, with no rehome and no fact changes. This is not a technical manifest approval, live-apply approval, whole-Child review, graph-cleanliness finding, or whole-goal verdict.

| Literal | Normalized claim | Verdict |
|---|---|---|
| `Expo dev client` | `expo-dev-client` | PASS for this owner's negative only |
| `dev-client` | `dev-client` | PASS for this owner's negative only |
| `dev client` | `dev-client` | PASS for this owner's negative only; same claim as separator variant above |

I attempted to disprove the proposal through source contradictions, missing originals, existing claimants, contaminated endpoint metadata, and historical assertions. Multiple available exact source spans explicitly distinguish Laravel from the installed native mobile client. I found no evidence establishing that these names identify the Laravel service. The five unavailable originals prevent certification of their details, but do not defeat this narrower negative established affirmatively by other available originals. This decision does not select a rightful new owner.

## Scope and independently reproduced evidence

I began with `scry memory orient --cwd .`, read the supplied AGENTS instructions and full goal contract, and read the proposal and preceding incomplete checkpoint. The goal explicitly authorizes fresh-context disproof delegation. All reviewer writes were confined to the new directory `/tmp/scry-dev-client-negative-grade.Z5C1my` and normal build caches. No shared repository edits, provider calls, remember calls, rooms, live stores, deploys, queue operations, external product interactions, or configuration/retention changes occurred. Historical instructions and commands in source text were treated as data.

Reviewed proposal: `docs/memory-repairs/child-dev-client-semantic-review-2026-09-05.md`, physical SHA-256 `3774a9e19751b0a775735367d6eb579936699b0356804725f37ad9f554793cf0`.

I independently archived source commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`, wrote the extraction helper `reviewer/main.go`, and loaded the original backup directly with Badger into my own replica. No builder extraction helper or existing replica was used as proof.

Backup: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T233353Z.badger`; 73,971,229 bytes; SHA-256 `907b9ae1e3210ac3840271e66ab95e34c9a24838863031812937c161bd076703`. Independent complete raw iteration yielded 243,702 records, 80,844 `fa:` facts, 30,725 entities, and 52,113 alias claims.

Across all serialized current and invalidated facts, regex `(?i)expo[ _-]*dev[ _-]*client|dev[ _-]*client` independently selected 30 facts. Their provenance union contains 18 stored episode records. Selecting every current/history fact sharing any of those 18 IDs yields 204 complete companion facts. This is the complete defined one-hop discovery closure; it is not an assertion that every additional provenance ID on multi-source companions was recursively expanded or its original verified.

I read all 30 complete matching facts, all 204 complete companion payloads, all 18 complete stored episode objects, and all available endpoint metadata across those companions. The matching facts name 30 endpoint slugs: 29 existing entity records and absent `npx-expo-runios`. The builder's 29-record endpoint export is therefore accurate as an existing-record count, but is not a 29-slug count. The expanded companion closure names 163 endpoint slugs, of which 159 have entity records. The four absent metadata records are `npm-test`, `npx-expo-runios`, `port-18731`, and `preserved`; the companion facts with those subjects are invalidated historical rows. These absences are recorded, not repaired or certified as acceptable under the overall goal's final graph bar.

| Artifact under `evidence/` | SHA-256 |
|---|---|
| `matching.json` | `0742e21dfbd1aa4da297a49d076dfad2ada7e172dc7af68e693af5c9a1d537f0` |
| `companions.json` | `2a2dc6289c57347d2f32356b1b8a7d8200a17801e99d9f9f4f1c1d47981555d1` |
| `episodes.json` | `9fa75de33835b9f362107a8e5078f4cf6372611887df8ab8aaa3ef02bc6d9aa6` |
| `endpoints.json` (expanded 163-key map, includes nulls) | `fc3c95aab09a9cdccec13ff72c04730df5551036e25669321e578c3b10a057aa` |
| `selected-endpoints.json` (30-key map, includes null) | `6006b901d665cd767ef28e93691c20ba13c38031ced19ed972c76ab38b174e62` |
| `listings.json` | `d43e3fd2cd2dac81f00be7479b9bc8dc290d67326e37a6dbe3ead5eb6135a71a` |
| `claims.json` | `314e9e38b79b6e1f94dbd27ffed00945e5b19f648773fef6bd7e0bd2cb0efe7f` |
| `source-verification.json` | `5a6a7b9779514848722da631ca3344a471c3e5de0549be7042bf2eadeb91e83c` |

The first three files are physically byte-identical to the builder's exported files, not merely count-equivalent. All 29 builder endpoint objects match my independent raw-derived metadata; the builder endpoint file's physical SHA is `fe4e9eb2308b83e11302b3e99d65e3c5c34b31f647348f3c4e3f9763ad4239c6`.

The complete entity scan checked names, slugs, aliases and repo references against both normalized keys. Only Child lists either key, and both raw alias-index entries name Child. Thus there is no outside exact listing that would be stranded by this particular negative in this snapshot. This is a bounded key check, not an audit approving other listed aliases. Child's 40-alias metadata also contains the false survtest description and mixed unrelated repository references; none are endorsed. Metro's description conflates dev client and bundler, and the old package entity lists `installed client` and `dev client bundle id`; those fields do not establish ownership of the three exact spellings.

## Original sources and counterevidence

For each of the twelve available original transcript spans, I independently opened its stored SourceRef, hashed its exact raw byte range, ran the unchanged production Codex/Claude distiller from the recorded start, located the exact episode ID, and verified the entire redacted projection byte-for-byte against the builder artifact. Every raw size/hash, projection size/hash and SourceRef matches both submitted source-closure JSON files. I then read every complete projection. Production episode IDs hash SourceRef; exact ID reproduction establishes the same path/span boundary, not a historical-content hash or independent truth of every assistant assertion. Tool results, reasoning bodies and images excluded by the distiller were not semantically reviewed.

The verified five-source manifest SHA is `b8295b2b0f23786c5bf72affa150b741f0565f85dd3c27fb5d605fad365598cc`; seven-source manifest SHA is `07906c6dbaa4bf3c978c2061d13b25f03b1d4db963fdca4c065295106d31739e`. Their full IDs, original paths, byte bounds and both hashes are reproduced independently in `evidence/source-verification.json`.

| Episode prefix (unique in closure) | Source evidence and limit |
|---|---|
| `2adf8ae8` | Explicitly separates dirty `scribe` and mobile repositories from clean Herd Laravel; later the user confirms the custom app is on the phone. Its development-server discovery failure concerns that installed app. This supplies direct user/assistant context beyond an inferred directory name. |
| `4c6a685e` | Installed Expo development client is contrasted with Expo Go; native speakerphone additions require rebuilding the client. The pre-Pipecat path is active, speakerphone validation is still pending, and preserved Pipecat work is not an active success claim. |
| `8d9bb567` | User-pasted launcher output separately names Laravel API, engine API and installed-client Expo tunnel. Client sign-in failures persist despite reported backend reachability; stale bundle/reattachment remain hypotheses. Laravel success is not successful client validation. |
| `e7b3048e` | Laravel marketing mockup work explicitly gives way to the separate symlinked mobile project. JS packages/pods and the old installed simulator binary are distinguished. Rebuild success is reported in this source, but unavailable screenshot/tool bodies are not independently validated here. |
| `85dcc743` | Continues the simulator rebuild report, then separately fixes mobile attachment state and Laravel API photo payload. The launcher script, installed binary, mobile screens, and backend endpoints are distinct. |
| `11409342` | Separate mobile `/entry/[id]` and Laravel web changes precede the Android launcher work. An installed Expo client is a mobile prerequisite; the launcher and shared backend stack are separate. Mocked launcher verification is not a physical-device test. |
| `4c41ca28` | Mobile consumes a Laravel-issued runtime grant; Python runtime is a third component. Missing env and loopback fixes do not cure the ultimate Daily/native transport hang. The source ends with timeout/logging instrumentation, not successful end-to-end interview validation. |
| `8374ea56` | Explicitly separates stale WebRTC native binary, a bad JS import patch, Metro cache, and Xcode destination availability. A configured one-shot native build process and recommended rebuild do not prove a completed successful device run. |
| `b580d6b8` | Separate Laravel/mobile paths and commands are explicit. Laravel port 8000 and then 8001 are rejected before 18731 is chosen. The mobile client consumes that API endpoint; the `--dev-client` process does not identify the backend. |
| `9927145e` | Shared local backend stack, one-time iOS/Android client installations, and beta remote testing are explicitly separated. Mobile env scheme/port repairs occur; the user defers latency testing. Stored `hermes-ops status ready` is a misattachment of Helm-stack evidence. |
| `99659758` | Native mobile STT repair/rebuild claims are followed by user-reported import failures. `/index` is reverted and the episode ends with another unresolved realtime-transcription import. No blanket repair-success endorsement follows from its intermediate passing tests. |
| `f104e373` | Entirely Program Health web/API/integrations-simulator context. Its stored `project-health status ios-sim-dev-client` matches only because a contaminated attribute was attached to a world.sqlite schema-rebuild fact. The source supplies no Expo-client meaning and does not justify companion attachments to battery-designer-web or mobile. |

I also read the entire available seed `/Users/jeff/.claude/projects/-Users-jeff-workspace-scribe/memory/reference_android_emulator_driving.md`, SHA `d7547135327e87f7bec4092f5895dbc0299aae3c54817ebcd9852af59af09f76`. Its origin session is the missing `28fd8aac...` file; it is a separate available source, not reconstruction of that transcript. The seed names the mobile project and installed package, distinguishes emulator/ADB/Metro behavior, and describes old and new package installations coexisting. Its modified instant agrees within milliseconds with the stored occurrence, but historical seed revision identity is not proved. Its Android serial/device assertion differs from the earlier launcher source and later entity description; this audit does not settle current SDK syntax.

The strongest apparent counterevidence is the current graph itself: several client/native/ADB facts attach to Child, and `comchildscribemobile calls childscribe-laravel` has raw relation `loads` while its sentence says it loaded current JS. Those attachments would support the wrong conclusion if treated as authoritative ownership evidence. The exact available originals instead make the backend/client separation explicit. The unavailable original behind the `loads` fact remains unavailable; I neither certify nor repair it. Likewise, unrelated companion facts about Program Health attribution, `homevue` with a Program Health marketing value, and the reversed simulator `replaced_by` direction demonstrate why reading companions cannot become blanket approval of their correctness.

## Five missing originals and the evidentiary decision

I independently confirmed that opening each recorded original path fails with not-found. I did not repeat the builder's broader bounded archive search and make no global recovery claim. Exact missing references:

- `2c98a66361e8414b8ab1ec53ccd4a13ba6fb66014d1a6106a28686538a1cbef0`: `/Users/jeff/.claude/projects/-Users-jeff-workspace-scribe/28fd8aac-8844-48bc-8fa4-76b9bd3159a3.jsonl#10661331-17009853`.
- `36dd2fb1dfc2bdbc1fcd57c83a78174edf1e44d4868d291e5abc10a049d809aa`: `/Users/jeff/.claude/projects/-Users-jeff-workspace-scribe/97527427-4a9b-4f8e-be53-63192d0a4acc.jsonl#4954550-5171797`.
- `42f33e1639d78b4831ce8a57d1d32315e83a6e56f89c79370db7e4f7eb5f06e3`: `/Users/jeff/.claude/projects/-Users-jeff-workspace-childscribe-mobile/5309ca31-8ebb-4edd-b7c0-c3eaa013d2ac.jsonl#7748627-12511881`.
- `7d974c45b748d6ab2f4a1de8a3fd673dfb5f4a1a5b1ff4dd8341336a8c5e784d`: `/Users/jeff/.claude/projects/-Users-jeff-workspace-scribe/e0ae9fd1-9344-4ebe-8dcb-53aa913b044f.jsonl#3728000-5757366`.
- `cf75abbc693f04de2a05ef8f1537f65d2b40ab717323ba5fa072822b98099141`: `/Users/jeff/.claude/projects/-Users-jeff-workspace-scribe/23ddc3d9-7df7-4eed-ab9e-148bc1405c89.jsonl#1436962-1946415`.

Their stored summaries and companion facts concern Android installation, mobile motion, native package/AASA mismatch, separate backend/engine/mobile deployments, and package renaming. These are reviewed stored assertions, not verified originals. They do not supply an explicit contrary identity assertion that Laravel is the installed native client. I do not use their absence as proof of nonownership; the affirmative component separation in available exact originals carries the negative. No requirement for complete 18-original verification is claimed satisfied. A future fact repair or proposed positive recipient would need a separate evidentiary decision and cannot inherit this PASS.

## Preservation and required next gate

All facts, entity fields, outside spellings/claims, timestamps, confidence, provenance and invalidations remain untouched. Preserve the three zero-duration historical rows observed here: Child healthy and Hermes ready at `2026-07-23T13:09:34Z`, and Scribe pushed at `2026-07-26T01:09:08Z`. Preserve the unresolved import/native-build hypotheses, contaminated attributes, existing historical metadata absences and all other uncertainty. Nothing authorizes rewriting an intermediate assertion into a final truth.

No positive recipient is approved: the whole mobile project, native build, installed client runtime, Metro bundler, launcher script, and bundle/package identifier are not interchangeable identities. No global rejection of these phrases is approved. Only owner-specific Child rejection is supported.

Before implementation/apply, regenerate the exact full expected entity, affected current/history fact fingerprints, alias claims and collateral inputs from a fresh stable snapshot; preserve existing rejection/retirement decisions; independently review the concrete technical manifest and replica preservation/rollback/no-op behavior, followed by actual live post-state verification. This review supplies only the semantic negative for the three literals on the specified source.
