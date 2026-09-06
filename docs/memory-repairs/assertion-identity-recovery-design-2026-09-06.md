# Assertion identity and historical recovery — draft

Status: design for independent disproof, not implementation or a live manifest.
The deployed24eafab guard remains active. Do not unpark conflicted inputs,
rewrite storage keys, restore an older full store, or recover any record from
this document alone. No credential-bearing historical cleanup is included.

## Evidence and required outcome

An old historical assertion and a different incoming assertion occupied one
normalized `(src, relation, KeyDst, valid_from)` storage key. The old statement
was overwritten before prevention deployed. Both exact originals remain in
separate complete backups, with provenance episodes available. Their values
normalize together but their full literals and statements differ.

PutFact now refuses that replacement atomically. However, earlier canonical
current-triple coalescing can still discard a distinct incoming sentence before
PutFact; mergeFact can also replace its start time with an earlier one. Merely
adding a storage suffix does not fix either resolver behavior.

The intended outcome is independent addressability of distinct assertions,
without invented timestamps, endpoint changes, dropped content or retained
transcripts. A reviewed recovery can then add the exact missing historical
assertion while preserving the replacement and every intervening record.

## Candidate design to disprove

1. Define a versioned assertion identity from length-delimited full source,
   relation, destination, literal value, raw relation, statement and UTC start
   instant. Do not derive it from clipped recall text, slugified literal value,
   delimiter concatenation or mutable confidence/provenance/InvalidAt fields.
   A hash match never substitutes for comparing the complete identity tuple.
2. Prefer one explicit versioned key format to permanently maintaining two
   implicit interpretations of the same tuple. Both forward and reverse keys,
   exact invalidation/deletion, store iteration, events, lexical index keys and
   recall deduplication must address the same assertion. Tuple-only references
   must refuse ambiguity instead of selecting one statement arbitrarily.
3. Migrate legacy raw fact payloads byte-for-byte, not by unmarshalling and
   reserializing them. Compute the exact full key/index delta in a restored
   replica; preflight malformed rows, duplicate destinations, all current and
   historical assertions, unknown fields, existing dangling endpoints and all
   nonfact families. Preserve existing defects visibly; do not infer owners or
   conceal them as migration cleanup. No unrelated raw family may change.
4. Introduce a fail-closed writer/schema floor BEFORE admitting new-format data.
   Prove that old binaries refuse it without modifying the store. A retained
   old binary alone is not a rollback after a format transition. The reviewed
   rollback plan must preserve and reconcile every intervening write, or keep
   the compatible writer while reverting only the faulty behavior. Do not ship
   a migration whose only fallback silently restores an old snapshot.
5. Change normal resolution to retain distinct incoming assertions rather than
   merging by a normalized current triple. Exact restatement can append source
   provenance under an explicit metadata policy, but cannot rewrite statement,
   full literal, raw relation or start instant. Earlier evidence is a separate
   assertion, not permission to backdate and erase an existing row. Existing
   supersession references need an explicit ambiguity policy and fixture proof.
6. Extend every reviewed merge, retirement, alias operation and raw snapshot
   fingerprint to preserve/assert the new identity and all exact reverse links.
   A graph repair may relocate reviewed endpoints but must retain the original
   assertion's other fields and predict its new address. No silent hash-based
   deduplication, timestamp nudge or guessed owner is allowed.
7. Only after the format/resolver source, full fresh replica and exact build
   survive independent disproof may prevention be deployed. Then regenerate a
   SINGLE missing-assertion recovery manifest from fresh complete live content
   and the verified old backup. It must pin both original assertions, original
   provenance, current replacement, full affected keys and all expected delta.
   Default dry run; fresh nonempty backup; atomic add; exact history and lookup
   proof; unchanged surrounding records; five suites; second no-op; independent
   actual proof. Do not overwrite the current replacement or replay its source.

## Concrete rejection tests before promotion

- Two normalized-equal literal values, identical triple/start, different text;
  current/current, historical/current and historical/historical combinations.
- Same literal and time but different raw relation or sentence; equal instants
  with different timezone spelling; identical full assertions with new provenance.
- Long values sharing200runes, delimiters inside text/literals, repeated or
  adversarial digest fixtures, malformed/unknown fields and absent/empty values.
- Every old full raw fact survives format migration byte-for-byte; all reverse
  links and exact read/invalidations find the intended assertion only.
- Whole-episode rollback, queue preservation, normal extraction replay on a
  complete replica, legacy writer refusal, interrupted migration and every
  backup/write/sync/close failure boundary. No partial schema transition.
- Existing rejection/retirement markers, complete historical group merges,
  second-pass no-ops, five unchanged question files and fresh independent recall
  grading after the last ranking/shape change. No expectation edits for gains.

## Open decisions, not delegated authority

The one-format proposal is unreviewed and may be more disruptive than an
explicit optional assertion discriminator with a writer floor. The reviewer
should disprove either design against the current code and identify the smaller
safe change; neither alternative is approved here. Current provenance/confidence
update policy and semantic supersession must be stated, not hidden in an ID.
Recovery of the one historical assertion is within reviewed preservation work;
credential deletion/redaction/rotation is a distinct authorization boundary.
The active goal and its two complete fresh grading rounds are not weakened.
