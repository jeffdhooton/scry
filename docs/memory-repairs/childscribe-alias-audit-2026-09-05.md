# ChildScribe alias audit — 2026-09-05

Audit/proposal only. No store repair, dry-run manifest, live write, deployment, model/API call, or remember was performed. The source backup was independently restored into this archive’s audit-replica directory.

Source: `/tmp/scry-qwen-stable-sep05.3oLvJ7/live.badger`

Backup SHA-256: `853d134fd52d4379d959c35181c5969d20ffa962c7df706a96cca8d4f9d26021`

Archive commit: `1af0d3bdb7019d4045e3512dde702f4fe74276f1`

Entity fingerprint (compact store Entity JSON SHA-256): `b5506902514ebf55037b57c9f67de04fe7f90af86a60471be60b517d6730ed5d`

Enumerated **92 aliases**, 82 exact normalized index keys. Decisions: {"drop":47,"keep":14,"rehome":7,"ambiguous":24}. All 2091 touching facts (including 522 invalidated) preserved as evidence. The snapshot has 30169 entities and 79547 total facts.

## Scope and evidence discipline

The prior reviewed prune 122→89 is not reversed. This snapshot holds three additional precise ChildScribe spellings (rows 90–92). Matching fact text was used to discover evidence, never as automatic identity authority. Candidate identities were inspected against facts and recorded in full. All rows carry exact index owners and outside normalized name/alias listings. Zero literal text matches are explicitly visible and do not establish alias provenance. Some keep decisions are stated semantic inferences from product/component facts, not literal alias attestations.

`childscribe-entity.json` contains the full exact entity snapshot. Its description is incorrectly “Default branch of survtest; unborn because no commits exist.” Its six repo refs include hoopless_crm, Docket, cleaning-company, Scry and Scribe; none is the attested ~/Herd/childscribe checkout. Alias repair alone cannot certify this entity or reattach its contaminated facts.

Read context: docs/MEMORY_HANDOFF_2026-09-04.md in full; DECISIONS entries on legacy routing, exact mention routing, fingerprinted merge, and alias retirement disposition. No new lexical rule, fact-count winner, or entity merge is proposed.

## Index obligations and unresolved work

- Rows 12–13: `forge` already lists both stolen aliases. Explicitly rehome index keys to `forge`; do not leave an unindexed rightful listing.
- Rows 55, 63, 66: exact keys already index `api`, which lists all three. Drop ChildScribe listings only and preserve those keys. API facts contain Docket and Program Health; the candidate is not certified clean.
- Row 46: `cleaning-company-marketing-site` lists `public marketing site`, while ChildScribe owns its index. No touching facts establish that outside entity; phrase facts identify Docket too. This row is **ambiguous**, requiring an explicit owner or reviewed removal of every listing. No unalias-only operation is proposed.
- Other rehomes are semantic proposals to existing entities that do **not** currently list the spelling. They require a reviewed mechanism supporting both target listing and index change; do not turn this table directly into standalone unalias inputs. Framework-version (#27) and short migration (#83) aliases also require the stated granularity/scope review.
- Domain/test/office duplicates, native dev-client identity, constructor function identity, misspellings, and workspace history remain unresolved as marked per row. None of the ambiguous spellings is approved to remain as a Laravel alias; the disposition means pause that mutation until it can preserve a justified lookup.
- Ordinary drops clear only ChildScribe-owned keys with no outside listing; normalized duplicate rows must be removed together. Their scoped candidate names remain independently indexed.

## Every alias

| # | Alias | Decision | Exact index owner | Existing candidates / targets | Outside listings | Rationale |
|---|---|---|---|---|---|---|
| 1 | Laravel backend | drop | childscribe-laravel | none established | none | Unqualified backend role also denotes Cell Saviors in the battery-designer fact; no global ChildScribe identity. |
| 2 | childscribe-backend | keep | childscribe-laravel | childscribe-laravel | none | Specific ChildScribe backend spelling. Laravel actions, Filament, and mobile integration facts identify the backend; no exact-text attestation for this hyphenation. |
| 3 | Laravel repo | drop | childscribe-laravel | none established | none | Repository role, not a unique project name. ChildScribe demo facts explain local usage but cannot make every Laravel repository this entity. |
| 4 | Laravel app | drop | childscribe-laravel | none established | none | Generic application role. Facts explicitly call Advocates a Laravel application too. |
| 5 | Lararel Forge | drop | childscribe-laravel | forge | none | Unattested misspelling of Forge, a distinct deployment platform. No exact fact attests this spelling; do not invent a new Forge alias from the typo. |
| 6 | /Users/jeff/Herd/childscribe | keep | childscribe-laravel | childscribe-laravel | none | Exact Laravel checkout path explicitly locates FamilyNewsletterIssueMail and agrees with the recorded apps/ symlink. |
| 7 | childscribe.com (staging) | drop | childscribe-laravel | childscribe-production, child-scribecom | none | Incorrect environment label: release-path facts separate childscribe.com production from child-scribe.com staging. Preserve correct domain names elsewhere. |
| 8 | Forge dashboard | rehome | childscribe-laravel | forge | none | Facts about Forge dashboard logs and queue-daemon configuration are platform administration facts on forge. Target currently does not list this spelling; proposal needs reviewed metadata/index introduction. |
| 9 | childscribe-laravel.on-forge.com | rehome | childscribe-laravel | childscribe-staging | none | Forge fact explicitly equates this hostname with the staging droplet serving child-scribe.com; childscribe-staging facts identify the staging web box and npm-cache deploy race. Host identity should not alias the Laravel service. Target does not yet list spelling. |
| 10 | https://child-scribe.com | ambiguous | childscribe-laravel | child-scribecom, childscribe-staging | none | URL names staging domain; child-scribecom is the direct domain candidate, but that entity also lists production childscribe.com and carries production facts. Resolve domain/host distinction and candidate contamination before transfer. |
| 11 | laravel backend | drop | childscribe-laravel | none established | none | Same normalized spelling and same cross-project evidence as Laravel backend (#1). |
| 12 | Laravel Forge | rehome | childscribe-laravel | forge | forge | Existing Forge identity lists Laravel Forge; independent Advocates and Cell Saviors deployment facts establish it is the deployment platform. Move exact index from ChildScribe to forge while removing ChildScribe listing. |
| 13 | forge-ssh-access | rehome | childscribe-laravel | forge | forge | Forge already lists this exact alias alongside Laravel Forge. Platform/server-access meaning and Forge deployment facts support retaining it there; rehome index to forge, never clear it under a surviving Forge listing. |
| 14 | prod web | drop | childscribe-laravel | none established | none | Unqualified production web role, not globally unique. ChildScribe health fact supplies context only. |
| 15 | childscribe.on-forge.com | rehome | childscribe-laravel | childscribe-production | none | Forge fact identifies production childscribe.com droplet with /home/forge/childscribe.on-forge.com; separate ChildScribe production entity has Stripe cutover/verification facts. Environment hostname belongs with production identity, not all Laravel code. Target does not yet list spelling. |
| 16 | childscribe-prod | ambiguous | childscribe-laravel | childscribe-production, childscribe-prodenv, childscribe-engine-core | none | Matches are mostly childscribe-prod.env and engine configuration; possible production environment shorthand, but bare spelling is not attested as the Laravel app. Do not choose from lexical similarity. |
| 17 | childscribe-app | ambiguous | childscribe-laravel | childscribe, childscribe-mobile, childscribe-laravel | none | Unqualified ChildScribe app can mean the product, native mobile client, or browser app. Sole substring hit is an App Store document; no unique backend referent established. |
| 18 | childscribe web frontend | keep | childscribe-laravel | childscribe-laravel | none | Product-qualified web frontend corresponds to Laravel/Vue browser app; admin-dashboard containment and ChildScribe Laravel web-app fact support the distinction from native mobile. No literal exact-phrase attestation. |
| 19 | ChildScribe domain | ambiguous | childscribe-laravel | childscribe, child-scribecom, childscribe-production | none | Product domain is not the Laravel application; staging and production domain identities are separate and the direct domain candidate is itself polluted. |
| 20 | staging Laravel | drop | childscribe-laravel | childscribe-staging | none | Unqualified environment/framework role. Staging Laravel facts describe ChildScribe deployment in context, not a unique global identity. |
| 21 | Laravel staging | drop | childscribe-laravel | childscribe-staging | none | Unqualified environment/framework role; deployment fact alone does not make all Laravel staging the ChildScribe app. |
| 22 | prod Laravel | drop | childscribe-laravel | childscribe-production | none | No exact-text attestation; unqualified production/framework role is not a unique global identity. |
| 23 | childscribe Laravel app | keep | childscribe-laravel | childscribe-laravel | none | Explicit product and component identity; admin-dashboard and Filament containment plus Herd symlink facts directly attest it. |
| 24 | laravel-backend | drop | childscribe-laravel | none established | none | Unqualified framework/backend role with no literal text attestation; equivalent ambiguity to Laravel backend. |
| 25 | ChildScribe backend | keep | childscribe-laravel | childscribe-laravel | none | CreateSocialPostDraft is a Laravel action in ChildScribe backend and task-1-brand is explicitly a Laravel backend task. Bare ChildScribe project remains separate. |
| 26 | Laravel API | drop | childscribe-laravel | none established | none | Generic API role. Facts describe both Cell Saviors Laravel API proxy and ChildScribe local API. |
| 27 | Laravel 13 | rehome | childscribe-laravel | laravel | none | Facts explicitly say ChildScribe web app is built with Laravel 13 and point to framework entity laravel. This names a framework version, not the app. Target does not yet list spelling; version-specific identity policy must be reviewed before execution. |
| 28 | web client | drop | childscribe-laravel | docket-web, web | none | Docket mount-table, driver-core extraction, and dispatch-client facts demonstrate cross-product generic client role. No global app identity. |
| 29 | staging domain | drop | childscribe-laravel | child-scribecom | none | Unqualified domain role. Facts supply child-scribe.com as scoped staging value, not an alias for Laravel code. |
| 30 | /Users/jeff/Herd/ChildScribe | keep | childscribe-laravel | childscribe-laravel | none | Case variant of exact Herd checkout in #6; documented macOS case-insensitive path and symlink resolution identify same checkout. |
| 31 | Repo: /Users/jeff/Herd/childscribe | keep | childscribe-laravel | childscribe-laravel | none | Annotated form of attested exact Herd checkout (#6). Prefix adds no different identity; no literal fact for annotated spelling. |
| 32 | https://childscribe.test | ambiguous | childscribe-laravel | childscribetest, childscribe-test | none | URL is local test deployment. Existing childscribetest and childscribe-test both carry test-instance facts; select/consolidate rightful identity before transfer rather than using application as host alias. |
| 33 | local ChildScribe app | keep | childscribe-laravel | childscribe-laravel | none | Specific local ChildScribe application; Herd-serving and local app symlink facts identify this checkout. No literal phrase attestation. |
| 34 | /Users/jeff/workspace/childscribe-laravel | ambiguous | childscribe-laravel | childscribe-laravel | none | No fact attests this workspace checkout path. Verified path is ~/Herd/childscribe; a historical checkout might be legitimate, but similarity does not prove it. |
| 35 | childscribe backend | keep | childscribe-laravel | childscribe-laravel | none | Case variant of product-qualified backend (#25), directly supported by Laravel task/action facts. |
| 36 | ChildScribe Laravel app | keep | childscribe-laravel | childscribe-laravel | none | Case variant of explicitly attested ChildScribe Laravel app (#23). |
| 37 | laravel app | drop | childscribe-laravel | none established | none | Same normalized generic Laravel app spelling as #4; Advocates provides direct counterexample. |
| 38 | ChildScribe Laravel backend | keep | childscribe-laravel | childscribe-laravel | none | Direct facts describe deployment on Forge and implemented memory-sharing feature in ChildScribe Laravel backend. |
| 39 | Laravel marketing | drop | childscribe-laravel | childscribe-marketing, childscribe-public-site | none | Marketing subsystem role, not entire app identity; candidate ChildScribe marketing entities exist but this unqualified framework phrase is not globally unique. |
| 40 | deploy-on-push | drop | childscribe-laravel | none established | none | Deployment behavior, not application identity; no literal fact attestation. |
| 41 | production posture | drop | childscribe-laravel | none established | none | Operational state/posture; actual matching facts are login/badge rehearsal posture on pnpm-bar-prod. |
| 42 | mobile/web side | drop | childscribe-laravel | web, childscribe-mobile | none | Cross-component area explicitly mapped by explorers, not a unique Laravel application. The existing web entity is itself mixed. |
| 43 | web voice | drop | childscribe-laravel | none established | none | Voice feature/surface: runtime-verification and photo attachment facts do not name the whole Laravel app. |
| 44 | childscribe domain | ambiguous | childscribe-laravel | childscribe, child-scribecom, childscribe-production | none | Same normalized domain alias as #19; cannot select production versus staging or product versus domain safely. |
| 45 | refs/heads/main | drop | childscribe-laravel | none established | none | Git branch ref. Exact matching fact locates it on Docket shared main worktree; not application identity. |
| 46 | public marketing site | ambiguous | childscribe-laravel | cleaning-company-marketing-site, docket-marketing-site, childscribe-public-site | cleaning-company-marketing-site | Outside cleaning-company-marketing-site lists exact alias but has no touching facts. Actual phrase facts identify Docket marketing too. Do not clear current index while outside listing remains: either explicitly rehome after provenance review, or review global removal from BOTH ChildScribe and cleaning-company-marketing-site. |
| 47 | marketing-site epic | drop | childscribe-laravel | docket-marketing-site, marketing-site-task | none | Facts explicitly place epic in Docket main/waves; task/epic is not Laravel identity and several Docket marketing-task candidates exist. Preserve their names, remove generic epic alias only. |
| 48 | forge user | drop | childscribe-laravel | forge | none | Unix account role on multiple servers; Cell Saviors trawl-install facts directly contradict Laravel app ownership. Platform is not identical to its account either. |
| 49 | Laravel web | drop | childscribe-laravel | none established | none | Generic framework/web surface. ChildScribe audit scope explains phrase locally but supplies no global unique identity. |
| 50 | envoyer | ambiguous | childscribe-laravel | cellsaviors, forge | none | Only fact concerns Cell Saviors Envoyer-style release deployment, not an attested actual Envoyer service identity. No envoyer entity exists; do not rehome a deployment tool name to the app or Forge. |
| 51 | office dashboard | ambiguous | childscribe-laravel | office-client, web-office-app, office, web | none | Clearly Docket office UI from dashboard Playwright evidence, but overlapping existing office identities need adjudication. Not a ChildScribe alias. |
| 52 | fleet worktree | drop | childscribe-laravel | none established | none | Many temporary fleet worktrees across tasks; exact ChildScribe-attached fact is a Docket worktree path. Generic worktree role cannot be a service alias. |
| 53 | Client Dashboard | drop | childscribe-laravel | childscribe-dashboard, office-client | none | Unqualified dashboard role, no exact text evidence identifying one product. Existing dashboards retain their own names. |
| 54 | client dashboard | drop | childscribe-laravel | childscribe-dashboard, office-client | none | Same normalized generic dashboard as #53. |
| 55 | api project | drop | api | api | api | Remove only ChildScribe listing; exact key already belongs to api and api still lists it. Preserve existing index. API facts contain Docket and Program Health contexts; this is not certification of api as a clean global identity. |
| 56 | loop orchestra | ambiguous | childscribe-laravel | setpoint, loom, setpoint-fleet | none | Substring matches are loop orchestrator, not exact loop orchestra. Setpoint orchestration is evidenced, but odd spelling and older Loom identity prevent attested alias transfer. |
| 57 | roll-off software | drop | childscribe-laravel | none established | none | Industry/software category; no exact text evidence. Does not name ChildScribe, nor does it justify selecting a roll-off vendor by word overlap. |
| 58 | driver-core worktree | ambiguous | childscribe-laravel | driver-core | none | Fact explicitly names driver-core worktree/branch on route-optimization. Package entity driver-core is related, not necessarily the checkout/branch identity. Do not attach worktree alias to package without review. |
| 59 | driver tree | drop | childscribe-laravel | driver-app | none | Fact names docket web/src/driver tree, a source area distinct from Laravel app. Generic tree phrase is not app identity. |
| 60 | areaScreens | ambiguous | childscribe-laravel | none established | none | Looks like source identifier, but zero exact fact matches and no exact existing entity. ChildScribe ownership unsupported; inspect source episode before selecting code identity or global drop. |
| 61 | office application | ambiguous | childscribe-laravel | web-office-app, office-client, web, office-app-shell-task | none | Office shell task and Docket web-entry facts establish non-ChildScribe surface; several overlapping candidate identities remain. |
| 62 | office service | ambiguous | childscribe-laravel | office-client, web-office-app, office-api | none | No literal phrase evidence; could denote UI or office API service. Remove-from-ChildScribe intent is clear, exact rehome is not. |
| 63 | createApp | drop | api | api | api | Remove ChildScribe listing while preserving existing api index/listing. createApp facts concern an app-constructor function, so api retention is only current-state preservation pending separate function/API ownership review. |
| 64 | fleet wave | drop | childscribe-laravel | setpoint-fleet | none | A wave is an execution grouping, not Laravel application identity. Numerous independent fleet-wave facts corroborate. |
| 65 | createApp() | ambiguous | childscribe-laravel | api, appts | none | Punctuated function spelling currently indexes ChildScribe, unlike createApp without parentheses (api). Facts explicitly name constructor in api/src/app.ts; a function identity is not API service. No attested exact existing function entity; do not infer ownership from punctuation fold. |
| 66 | api app | drop | api | api | api | Remove ChildScribe listing only, preserving existing api index/listing. Facts explicitly include Program Health API app as well as Hono/Docket contexts, so api needs separate identity audit. |
| 67 | campaign engine | drop | childscribe-laravel | claude-code, codex, setpoint | none | Actual matching facts name Claude and Codex as two campaign engines. Generic execution role cannot name one Laravel app or one unique rehome target. |
| 68 | stops table | ambiguous | childscribe-laravel | 0112-operations-routes-stopssql, stops | none | Evidence identifies database table created in migration 0112. Existing stops entity is a driver-core source directory, not table. Migration file contains definition but is not table identity; no proved rightful table entity. |
| 69 | fleet architecture | drop | childscribe-laravel | setpoint-fleet, setpoint | none | Architecture facts explicitly concern setpoint/scry/fleet. Architecture topic is not identical to service or repository. |
| 70 | fleet skill | drop | childscribe-laravel | room-worker, setpoint-fleet | none | Exact phrase fact invokes room-worker fleet protocol. Generic skill role can mean fleet or room-worker, and is not an app alias. |
| 71 | Expo dev client | ambiguous | childscribe-laravel | childscribe-mobile, expo | none | Facts explicitly require custom Expo dev client for ChildScribe mobile; tool/build is not Laravel, mobile app, or whole Expo toolchain identity. No exact existing dev-client entity found. |
| 72 | dev-client | ambiguous | childscribe-laravel | childscribe-mobile, expo | none | Native WebRTC dev-client binary and simulator facts establish mobile build, but no unique rightful dev-client entity. |
| 73 | development build | drop | childscribe-laravel | none established | none | Generic build flavor with no exact fact evidence; not app identity. |
| 74 | --dev-client | drop | childscribe-laravel | expo | none | Exact fact uses it as CLI flag in npx expo start command. Flag value is not Laravel, Expo, or mobile app identity. |
| 75 | dev client | ambiguous | childscribe-laravel | childscribe-mobile, expo | none | Same mobile native-build meaning as #71/#72. Metro/rebuild facts misattached to Laravel must not be used as alias identity authority. |
| 76 | iOS client | drop | childscribe-laravel | childscribe-mobile | none | Platform/client role. Native WebRTC evidence locates a ChildScribe mobile client in context; unqualified iOS client is not globally unique. |
| 77 | voice pipeline | drop | childscribe-laravel | childscribe-mobile, realtime-tool-call-loop | none | Feature/pipeline spans mobile and OpenAI Realtime, and later mobile/web exploration; not whole Laravel application. |
| 78 | child-scribe.com (staging) | ambiguous | childscribe-laravel | child-scribecom, childscribe-staging | none | Explicit staging domain name; domain entity is contaminated with production alias/facts and staging machine is not same abstraction. Same unresolved domain/host decision as #10. |
| 79 | production profile | drop | childscribe-laravel | easjson | none | EAS build configuration profile, not app. eas.json fact and mobile production-profile facts directly establish configuration context. |
| 80 | prod environment | drop | childscribe-laravel | childscribe-production | none | Unqualified environment role with no literal attestation. Retain specific production identity elsewhere. |
| 81 | marketing source | drop | childscribe-laravel | web-marketing | none | Actual fact describes source-code boundary in Docket marketing, not ChildScribe code identity. Generic source-area phrase. |
| 82 | loop engine (former) | ambiguous | childscribe-laravel | loom, setpoint, setpoint-fleet | none | No literal attestation. Could mean renamed Loom/Setpoint, but former label alone cannot adjudicate overlapping existing loop identities. |
| 83 | migration 0520 | rehome | childscribe-laravel | dbmigrations0520-solve-runssql | none | Direct route-solver-domain fact names migration 0520_solve_runs.sql and existing exact file identity; other route-solver facts require same migration. Target currently lacks short alias; scope short number to Docket provenance before execution. |
| 84 | scrible | ambiguous | childscribe-laravel | scribe, childscribe-monorepo, scribe-repo | none | Unattested typo-like spelling, no scrible entity. Scribe monorepo is separately evidenced; do not assign typo by lexical similarity or keep it on backend. |
| 85 | scrible-monorepo | ambiguous | childscribe-laravel | scribe, childscribe-monorepo, scribe-repo | none | No exact-text attestation for misspelled monorepo name. Existing Scribe monorepo candidates are distinct from contained Laravel app; need source provenance. |
| 86 | web frontend | drop | childscribe-laravel | docket-web, web | none | Facts explicitly say frontend is part of Docket repo, including misattached self-loops. Generic frontend role is not Laravel identity. |
| 87 | Envoyer release deployment | drop | childscribe-laravel | cellsaviors | none | Deployment pattern from Cell Saviors fact, not app identity; no actual Envoyer identity or exact spelling attested. Preserve fact for later endpoint review. |
| 88 | Laravel app symlink | drop | childscribe-laravel | appschildscribe-laravel, appschildscribe-laravel-symlink | none | Facts describe the apps/ symlink relation and target. Unqualified symlink role is not service identity; preserve exact existing path identities rather than adding generic path label. |
| 89 | Laravel Herd project | drop | childscribe-laravel | laravel-herd | none | Generic Herd-managed project class; facts explicitly state projects plural live under ~/Herd. Neither ChildScribe nor Herd tool owns every project. |
| 90 | jeffdhooton/childscribe-laravel | keep | childscribe-laravel | childscribe-laravel | none | Exact owner/repository identifier attested by GitHub push and PR URL facts. |
| 91 | childscribe-laravel repo | keep | childscribe-laravel | childscribe-laravel | none | Exact repository-qualified name directly attested by Filament placement and marketing fleet targeting ~/Herd/childscribe. |
| 92 | childscribe-laravel memory entity | keep | childscribe-laravel | childscribe-laravel | none | Explicit self-reference to this exact memory entity, not a different product. No literal fact attestation; harmless precise lookup spelling but optional to retain in a later reviewed metadata policy. |

## Artifacts

- `review-dispositions.json`: enumerated decisions, rationale, exact normalized keys, owners, outside listing snapshots, candidate target listing state and index disposition.
- `alias-fact-evidence.json`: all literal matching fact snapshots for every alias, including invalidation state, endpoints, and episode IDs. Substring matches are discovery evidence and must be read in context.
- `candidate-identity-evidence.json`: complete candidate entity snapshots, fingerprints, and all touching facts; preserves evidence of candidate contamination.
- `review-episode-evidence.json`: provenance episode metadata and summaries for matching/candidate facts. Original transcripts were not reread.
- `childscribe-entity.json`, `childscribe-facts.json`, `alias-index-evidence.json`, `snapshot.json`, `review-summary.json`: exact entity and graph evidence, counts and explicitly defined fingerprints.
- `cmd/aliasaudit/main.go`, `evidence.mjs`, `dispositions.mjs`: reproducible replica restore/export and report generation. Do not run the restore helper against an existing store directory.

No benchmark was rerun because no mutation was tested. This review does not supersede the earlier prune’s benchmark result and does not establish current live state, recall improvement, or final graph correctness.
