# Independent bounded semantic review — Child `stops table`

Verdict: **PASS**, solely for the owner-specific disposition removing the
`stops table` alias spelling and its `al:stops-table` claim from
`childscribe-laravel`, with durable negative evidence for that owner. No
positive owner is established. No fact, episode, other alias, entity
description, repo reference, or endpoint is approved for change.

This is a semantic verdict against the complete Mini backup from 22:31:22,
not an apply-ready manifest, complete-replica transformation gate, permission
to apply against a changed fingerprint, cleanup-completion claim, or claim
that the overall goal is complete. The independent technical gate must be
refreshed after the separately reviewed three-alias Child batch, including
its resulting Child fingerprint and three owner-specific rejection records.

## Independent evidence acquisition

I read both proposed review files completely. I created my own archive of
commit `d1f0a958608389a385ac9a9f57ec1eb941015a10` at
`/tmp/stops-independent.R5A31f` with `git archive`, authored the private
`cmd/independent-stops/main.go` helper, and restored both complete backups
directly with Badger `Load`. I did not use the builder's filtered exports as
proof and did not call any extraction provider, retry queue, installer, or
live mutation. All created files and replica databases are inside this
private directory. The user's untracked assessment document was untouched.

Verified source inputs:

| Input | SHA-256 |
| --- | --- |
| `mini-child-final-223122.badger`, 73,526,260 bytes | `b669593b978041a646b8c6f3f3dc2cee5eb8cbad0e83d48063915a7a7aaeaea0` |
| `mini-child-fresh-221012.badger` | `4a33e4d8b7e2786d3c9a936cf8bf71f7ca61603ecb8f3454b6f26a8dcb946382` |
| supplied deployed `scry` artifact | `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553` |

Fresh restore: 80,586 facts, 30,604 entities, 9,353 episodes, 51,957 alias
claims; 0 `ar:` rejection records. Earlier restore: 80,473 facts, 30,551
entities, 9,342 episodes, 51,889 claims; 0 rejection records. These are
complete stores, including historical facts, not current-only exports.

The helper iterated every raw key/value into `fresh/raw.json` and
`earlier/raw.json`, then decoded the entire fact/entity/episode families.
Their compact-JSON map hashes are respectively
`192ae88274b5686bae273435f9954414e3a3b2f435516108e3a2ece7e9f42aa6` and
`6a0817b2a0c09fc4f183cd59ec2abbf540098f26508cc460fba3912582210357`.
Physical file hashes (which include formatting/newlines and differ from
compact semantic hashes) are recorded in `file-hashes.json`.

## Attempts to disprove the disposition

1. **Direct table evidence.** The exact FK assertion is the current fact
   `0043-weight-ticket-task-linksql / references /
   0112-operations-routes-stopssql`, valid from
   `2026-08-19T05:59:21.935Z`, sourced to episode `7a57052a...`.
   Its complete reconstructed text describes the table referenced by
   `weight_tickets.stop_id`, with exact migration and disposal repository
   paths. The complete Docket six-task plan `4d8170c9...` contains this exact
   worker task, and the original final worker text `15057dba...` reports it.
   These are concrete table/migration/repository identities. They do not
   identify the whole ChildScribe Laravel service as a table.

2. **Discovery false positives.** A case-insensitive serialized search of
   all 80,586 current/history fact rows yields five `stops table` matches.
   Searching space/underscore/hyphen separator variants yields the same
   five. Three are the `recurring_plan_stops` contract dispute, not the
   exact generic table identity. The complete originals `0d77c4f6...`,
   `0b6898c5...`, and `50453783...` concern operations-domain review and the
   `linkStop` versus `recordOccurrence` boundary. They establish no Child
   identity. The remaining survey match `4181bcf3...` says explicitly
   `Repository: /Users/jeff/workspace/docket`; its full text discusses
   orders, tasks, routes and stops as separate schema deliverables.

3. **Actual claim and outside listings.** I checked every slug, name, and
   alias in the complete 30,604-entity inventory using the code's Normalize
   semantics (trim/lowercase, collapse runs of ASCII spaces/underscores to
   hyphens). Exactly one entity lists normalized `stops-table`: Child.
   The actual raw claim is `al:stops-table → childscribe-laravel`.
   Child currently lists 43 aliases; its description is the contaminated
   `Default branch of survtest; unborn because no commits exist.` and its
   repo references include unrelated projects. Those fields are not
   independent ownership evidence.

   I also inspected the names/aliases of all 174 broader `stop`, weight
   ticket, 0043 and 0112 discovery candidates, plus the complete entity
   metadata for the selected source closure. `en:stops` is named `stops/`,
   typed `machine`, described as the new driver-core domain directory, and
   lists `stops domain`, `stops`, `Stop`; `al:stops` points there.
   `en:0112-operations-routes-stopssql` is named
   `0112_operations_routes_stops.sql`, typed `tool`, and describes the
   migration block containing the FK-parent table; its actual claim is
   `al:0112-operations-routes-stops.sql`.
   Neither is interchangeable with the table on this evidence. The
   disposition must not rehome the alias to either, to Docket, or to any
   newly invented table entity.

4. **Contaminated companion counterclaims.** All 58 facts sharing the five
   matching source episodes were inspected, including current/historical
   rows, and all 87 facts sharing the original seven source episodes were
   read. Child-linked assertions include missing orders, task_types and
   tasks; the one-row exchange contract; the worker's owned-path assertion;
   a scale-ticket migration containment assertion; and 0042's frozen-path
   edge to Child. Their underlying original texts concern Docket work and
   do not support `stops table` as the Child service's identity.
   These endpoint assertions remain **UNRESOLVED for separate fact review**.
   No sentence-based repair, fact drop, invalidation, or reattachment is
   approved here. In particular, shared-episode membership does not certify
   every stored companion sentence: some accumulated fact texts concern
   other work, even when this episode remains in their provenance array.

5. **Contradictory implementation history.** The stored `7a57052a...`
   summary and companion claim a row-copy rebuild using
   `PRAGMA legacy_alter_table` / `sqlite_parse_foreign_keys`. Its original
   full text records FK-parent prepare-time failures. The full
   `15057dba...` final report, commit `d755529`, explicitly says the
   migration has **no row-copy statement**, and instead an empty-table
   guard aborts while retaining original rows if non-empty. The report
   also discloses three out-of-owned-set test files, contrary to the
   earlier all-owned-paths companion. I preserve these assertions as
   historical counterclaims rather than treating the earlier extracted
   summary as verified implementation truth. Broader existing metadata
   additionally records later 0043/0200 two-phase repairs; that later
   metadata does not erase the no-copy final report and is not independent
   proof of Child ownership or of the current implementation. Resolving
   the full migration history is outside this alias-only disposition.

6. **Additional Child-linked counterclaims independently expanded.** I
   reconstructed and read five further complete original projections:
   `fb0929fd...` (Docket Wave 25 survey and portal claims), `f9870282...`
   (Docket driver-core guard and parity discussion), `df7ddc01...` (Docket
   portal door and design query), `b5b0ae78...` (Docket Wave 23 survey), and
   `1b1070f7...` (Docket Wave 24 survey). They are the provenance for the
   remaining Child-touching weight-ticket/portal/OCR discovery claims.
   All 77 same-episode companion sentences were read. These sources add
   no Child table identity: the portal brief expressly concerns a dumpster
   hauling customer; the surveys identify the Docket repository; and the
   driver-core source describes a directory move independently of the
   operations database tables. Some stored companion conclusions exceed
   their source projection; those are also preserved and not adjudicated
   as correct facts here.

## Full original-source reproduction

The independent helper parses each original **whole session file**, selects
the exact stored episode ID, writes the complete reproduced text to a
private `.txt` file, and separately hashes precisely the original SourceRef
byte span. Whole-file parsing naturally reproduces the short `15057dba...`
tail without lowering the minimum-turn floor or fabricating a replacement
episode. The builder's failed first reconstruction directory was untouched.

All seven original raw-span lengths/hashes and all seven reproduced text
lengths/hashes match the proposed closure exactly. Exact IDs, original
absolute paths, byte spans, lengths and hashes are retained in
`sources/closure.json`, SHA-256
`393c36d749ba1525cd9f110a5f189651698acf294bc33b7f4714a8dfc1cb84dd`.
The five independent additional sources have the same complete provenance
fields in `extra-sources/closure.json`, SHA-256
`0dc478e2caf449d4bf096c0fb465f52d668ea170f8b977d64527086ad33e7076`.
All twelve `.txt` projections were read completely. This is not a claim of
manual reading of omitted raw tool-result/thinking payloads: the exact raw
spans are hashed, and deterministic distillation retains the original full
human/assistant projection, including any truncation already inside the
historical source itself.

## Fresh semantic drift against the 22:10 source

I independently restored both backups. Complete raw delta: 442 added keys,
38 changed keys, zero removed keys. Fact-object delta: 118 new/changed
versions versus five earlier versions, net +113 rows. Entity-object delta:
71 new/changed versus 18 earlier versions, net +53 entities. Exactly 11 new
episodes and no changed/removed episodes. `diff.json` retains all complete
old/new objects; `raw-delta.json` retains exact changed key lists.

The 118 fact sentences, their five old versions, all new/changed entity
metadata and all 11 new episode records were inspected. The additions
concern Scry deployment/review reports, CADFormats, cockpit and filing or
licensing tool planning. The Scry three-alias review is a report about a
review, not new domain ownership evidence. None introduces a new matching
table fact, a Child adjacency row, a competing normalized listing, or a
changed table/Child identity. The only changed entities within the expanded
164-fact endpoint closure are `codex-reviewer`, `jeff` and `scry`; their
updated metadata concerns those later reviews/projects and does not claim
the table spelling.

The following compact JSON hashes are equal across both independent
restores (preserved object field ordering, no pretty-print whitespace):

| Closure | Count | SHA-256 |
| --- | ---: | --- |
| Exact substring matching facts | 5 | `d5c0906dad0f2e55a77789799f4e820d71d41142aa7bac943de554153afc0799` |
| All matching-episode companions | 58 | `526643b8f4b34b69b17253b43610d84bf72d0e14d36274d91391f1cba7928be5` |
| Original seven-source fact closure | 87 | `2a5e8010307d8d4f83d6aa7c1de1684faa1f708c4e69d2e40727e79f6518b92b` |
| Expanded twelve-source fact closure | 164 | `d76ea10656fb14757c0166aa179716611b818e72a6e9255cb064d48955bcdc59` |
| Entire Child current/history adjacency | 2,106 | `fdcf195d2de089b776504b11a2a608763c009740fa0fb0a88e1cb14441b533de` |
| Child complete entity object | 1 | `b7fa1df19b48d1713368d546bda53d32ea6e748292092936fc10b55f26026c59` |
| All exact normalized listing entities | 1 | `58126f0d8a429535d280588e8f586908a6380d094183d65bc774e7f21c73481f` |
| Full selected episode metadata | 12 | `f00080790020a03adbe54ec21e065d0cbb25600c0c79c86894eec1e50c5dbf87` |

`semantic-comparison.json` and `file-hashes.json` distinguish reproducible
semantic hashes from hashes of the actual retained evidence files. No
blanket claim is made that all 2,106 Child edges are semantically sound.

## Scope of the PASS

The sole alias disposition has no unresolved semantic blocker on this
snapshot. Its owner-specific negative rationale should name the concrete
Docket FK table source and the migration/directory distinction, state that
no **outside** entity lists the exact normalized spelling, and avoid
claiming every companion assertion is true or no Child alias listing
exists. Preserve every current and historical fact unchanged.

The future live technical action remains **UNRESOLVED / not reviewed here**:
refresh its backup, complete affected/outside/rejection closure and exact
fingerprints after any earlier batch, obtain independent complete-replica
and immediate-live-input gates, use synced nonempty backup and atomic
guarded apply, then independently audit actual pre/post state, exact
lookups, required benchmarks and second no-write. This semantic PASS does
not permit global heuristics, blind alias backfill, the old 41-group apply,
timestamp nudges, or new retention policy.
