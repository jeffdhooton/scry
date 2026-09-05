# Independent canonical-name code/replica gate: PASS

Exact candidate `62cf6e0d2d8db9d324da23114df837848e972c8f`, independently archived into a new temporary directory on 2026-09-05. Neither rejected predecessor (`a965177`, `1dac187`) was deployed; live remains `393eeec` during this review.

This bounded PASS means the independent disproof attempt found no remaining safety regression in the canonical-name exception under the tested contract. It is not a deployment receipt, an assertion that the parked provider output was replayed, a recall-floor pass, or full-goal completion.

## Both earlier failures are closed

1. Two established canonical homonyms named Atlas under nonnatural retained project/machine slugs now return `ErrAliasClaimed` instead of assigning a machine fact to the project index owner. Whole facts/entities/episode/alias state is unchanged. Existing natural-slug exact-identity precedence remains intact.
2. Generic canonical references refuse mismatched identity metadata. The fixed spaced canonical owners `our own machine` and `this physical host` now refuse all incoming space/underscore/hyphen variants. The prior candidate accepted the four nonspace cases; the exact independent regression now passes.

The production correction applies the existing determiner check to the same normalized separator group used by canonical equality. It does not expand ordinary alias admission or introduce another word list.

## Additional independent probes

- **Transaction-visible ambiguity:** a second canonical homonym staged inside the active transaction is found by the complete entity scan. The resolver refuses and the transaction rolls back completely; it does not rely on a stale external inventory.
- **Normalization:** case changes, surrounding whitespace, underscores, and hyphens do not hide another canonical owner. The scan refuses any other canonical homonym, regardless of type.
- **Reference groups:** 24 additional cases cover eight phrases with space/underscore/hyphen mentions, including `the physical host`, `our own workstation`, `this remote server`, `your render worker`, `their observatory`, `some container`, and `its build daemon`. None changes a legacy project owner's identity metadata or aliases. Earlier generic skips are allowed only where no identity mutation occurs; typed refusals leave the whole graph unchanged.
- **Ordinary real identities:** 19 independently selected proper-name/artifact cases—including A-Frame, The Graph, Wasmtime, NATS, DuckDB, OpenTofu, Mistral-7B-Instruct, GNU Make, and a migration filename—retain existing unique identities under nonnatural slugs when declared with their existing type. No duplicates or description replacements occur.
- **Ownership and type guards:** stale index claims, incompatible noncanonical aliases, typed mentions through concept-only aliases, and punctuation/natural-slug collisions still refuse atomically.
- **One-episode resistance:** two distinct existing machine/project identities declaring each other's names as aliases keep their original index ownership and distinct fact endpoints.
- **Concept upgrade:** a unique retained canonical concept can upgrade to its supplied type while preserving its original description and identity; case/separator variants do not create duplicates.

## Actual postmigration backup probes

Source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger`.

SHA-256 independently verified before/after review: `5323fef7b594c748fd61462909cc808f78c1bd0a104da070abdc54cb21e102aa`.

Fresh independent restores contain **80,203 facts**, **30,436 entities**, and **51,749 alias-index claims**. The current reviewed SQL-file identity has canonical name `db/migrations/0160_task_evidence_rules.sql` and retained slug `0160-task-evidence-rulessql`.

Synthetic declarations typed `tool`, `concept`, `project`, `machine`, `service`, and protected SQL-artifact `value` all resolve to that retained identity. Each probe adds two explicit synthetic facts, using the canonical filename as both source and destination. The original 80,203 facts—including current and invalidated history and provenance—remain identical. Every alias claim remains identical. All entity metadata remains identical except the survivor's intended LastSeen refresh to the synthetic episode timestamp; reviewed type/name/description/aliases/repo refs/creation time remain unchanged. No natural-slug duplicate is created.

These are independently exercised matching mechanisms, not a reconstruction of missing provider output. **The actual extracted type of the real parked note remains unknown.**

## Runtime and remaining conservative limits

Forty full exceptional `resolveEntity` transactions were timed on a separate fresh restore and then deliberately rolled back. All graph hashes matched before/after.

- p50: **71.658 ms**.
- p95 (38th sorted sample): **78.984 ms**.
- Maximum: **120.806 ms**.

The work includes the transaction-visible scan over 30,436 entities and ordinary entity-update processing. This was measured on the review laptop while regression checks also ran. It is credible local cost evidence, not live remember p95. The scan is linear in entity count and is paid for each exceptional nonnatural canonical-name/type-bypass mention; a long episode with many such mentions can accumulate that cost. Ordinary compatible names and noncanonical aliases avoid it.

The exception is intentionally conservative, not universal semantic name recognition. A differently typed mention of a retained canonical name whose normalized words look like a reference—e.g. A-Frame or The Graph—still refuses atomically. Correctly typed mentions pass. The same mismatched cases already refuse on live-code `393eeec`, independently verified in `brand-limitation-old393.log`; therefore this is not a new production false rejection. The gate must not be summarized as “every possible mistyped canonical name now succeeds.”

## Verification and artifacts

Review root: `/tmp/scry-canonical-finalgate-sep05.uyfOSP`.

- `go test ./internal/memory/resolve -run '^TestIndependent' -count=1 -v`: PASS (`independent-tests.log`), covering both former failure corpora, staged ambiguity, six actual-store probes, and runtime preservation.
- `go test ./internal/memory/resolve -run '^TestIndependentFinal' -count=1 -v`: PASS (`final-boundary-tests.log`), covering the additional 24 reference cases, 19 real names, and two documented conservative limitations.
- `go test ./... -skip '^TestIndependent'`: PASS (`full-suite.log`).
- `go vet ./...`: PASS (`vet.log`).
- Focused independent race run covering staged/normalized homonyms, determiner normalization, original homonym refusal, and one-episode ownership resistance: PASS (`race.log`).
- Exact 393eeec differential for the two conservative brand/type-mismatch cases: PASS refusal/preservation (`brand-limitation-old393.log`).

Evidence SHA-256:

- `independent-tests.log`: `7a4e7206a795f39cb61c3ed1599aeff9a2c2aac43fab919078eb9df544e301c0`.
- `final-boundary-tests.log`: `8d5d7d60b989bf1fbe2f8aae4b130290ce7025c7240b9468d3e463a3432440af`.
- `full-suite.log`: `5873f1e3f6858ccde66cc23e03945a3be36e192a21dfe45dfdd732ad04de4ab9`.

The goal house rules were read earlier in this continuous independent review session. All code/tests/reports and synthetic store writes were confined to review-owned temporary paths. No shared-repository edits, live writes, deployments, model calls, remember calls, historical repairs, inferred owner transfers, or benchmark changes occurred. The exact code/replica gate is PASS with the stated scope and limits.
