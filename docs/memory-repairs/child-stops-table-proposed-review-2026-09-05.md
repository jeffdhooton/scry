# Proposed ChildScribe alias disposition: stops table

Root source review only, not an independent verdict or apply-ready manifest.
No live write, fact move, alias rehome or production-code change. Keep separate
from the already reviewed three-alias batch; do not enlarge that batch silently.

Source is the complete 22:10:12 Mini snapshot, SHA
`4a33e4d8b7e2786d3c9a936cf8bf71f7ca61603ecb8f3454b6f26a8dcb946382`.
Complete facts/entities/claims are retained at
`/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/child-three-fresh-221012/before-*.json`.
That source contains 80,473 facts and all 2,106 current/history Child-touching
rows. Proposed disposition: remove only Child's `stops table` spelling/claim,
with durable negative evidence for this owner. Do not invent a positive owner.

## Evidence and counterclaims

The full source projection for episode `7a57052a...` explicitly describes the
weight-ticket FK link to Docket's `0112_operations_routes_stops.sql` table.
Its implementation paths are `db/migrations/0043_weight_ticket_task_link.sql`
and `api/src/repos/disposal/weight-ticket.ts`; a table is not the whole Laravel
application. The existing `0112-operations-routes-stopssql` entity names the
migration file, not the table. Existing `stops` is named `stops/`, describes the
driver-core domain directory, and lists `stops domain`, `stops`, `Stop`. None of
these distinctions authorizes routing a generic table alias to either entity.

Case-insensitive serialized `stops table` discovery finds five fact rows. Three
are substring matches to the DIFFERENT `recurring_plan_stops` proposed table;
their full original episodes `0d77c4f6...`, `0b6898c5...`, `50453783...` concern
a Docket operations contract dispute and supply no Child ownership evidence.
The fifth source `4181bcf3...` is explicitly a Docket wave-planning survey;
its source text says Repository `/Users/jeff/workspace/docket`, with separate
operations, tasks, routes and stops schema deliverables. All five full distilled
texts and all 58 same-episode companion fact sentences were read, not merely
the stored summaries. Raw source spans were hashed; this does not claim manual
reading of omitted tool-result/thinking payloads or correctness of every companion.

Several companion assertions incorrectly attach Docket work to Child: orders,
task_types and tasks missing-table facts, an exchange contract, and the worker's
owned-file worktree assertion. The actual sources above do not identify Child.
Two further possible counterclaims were traced through complete sources:
`15057dba...` is the same weight-ticket worker's final report (d755529); it
describes composite task/stop FKs and the Docket migration paths.
`4d8170c9...` is the Docket six-task campaign plan containing that exact task.
Neither establishes Child ownership. Preserve these contaminated facts unchanged
as UNRESOLVED for separate endpoint review; do not repair them by sentence rules.
The final report also contradicts the earlier row-copy implementation summary;
both historical assertions must remain, not be silently collapsed or rewritten.

Every entity's slug/name/alias was inspected for this exact normalized spelling;
only Child lists it in this snapshot. This is an owner-specific negative proposal,
not a global ban on a table name. Child's polluted description/repo refs and all
remaining aliases are unchanged and not approved by this proposal.

## Reproduction and outstanding gate

Source hashes, exact refs and fact-closure hashes are in the adjacent
`child-stops-table-source-closure-2026-09-05.json`. Diagnostic helper:
`/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/code/cmd/alias-source-review/main.go`.
It writes only new private evidence directories, hashes exact original byte spans
without retaining new raw-span copies, and requires reproduced episode IDs.
Episode 15057dba has only two substantive turns in its tail slice, so starting
distillation at that slice hit the minimum-turn floor. Reprocessing the original
whole file reproduces the exact stored ID and the complete 3,108-byte tail text;
the failed first attempt is retained, not mislabeled a missing source or bypassed.

Before any apply: regenerate from a fresh stable backup AFTER the current batch,
include existing rejection records, review all fresh affected/outside/counterclaim
closure, obtain a fresh independent semantic and complete-replica verdict, check
exact live inputs, use synced nonempty backup plus atomic guarded repair, then
independent complete actual pre/post audit, exact lookups, all five benchmarks
and second no-write. This document alone authorizes none of those live changes.
