# Exact tie order: bounded reproducibility change

Status: bounded independent source PASS; NOT deployed. Full report:
recall-exact-tie-independent-review-2026-09-06.md, SHA
849f12ab0a2e555365e3b2a2803b9ebb3ead2f73f7507381026f7eebe4d92247.

The independent relative-reason disproof found two distinct full results in
thirty repeated identical baseline queries. Equal-score, equal-time cs-345
and cs-346 facts alternate as the surviving duplicate. Thirteen other
non-reason rows had candidate-tail differences while their top twenty stayed
the same. This predates the rejected reason-prior experiment.

The proposed change retains descending score and descending validity time,
then resolves exact lexical-search ties by document key before the candidate
cutoff. Recall resolves exact final-score/time ties by its existing fact-hit
identity key before duplicate suppression and diversity. Neither stage gains
a new scoring feature, weight, synonym, candidate count or question exception.
Entity headers already have their own slug tie-breaker and are unchanged.

The independent reviewer proved an important limit: recall's existing hitKey
can collide when toHit clips long values or delimiters are ambiguous. This
change orders distinct hit keys; it does not fix that earlier identity loss
or establish a total order for every possible stored fact. Unequal named
endpoint weights also affect scores before sorting, independently of this
change. Both issues reproduce on the baseline and remain separate work.

Measured evidence: clean no-CGO full tests and focused race pass; all five
suite hits, misses and mean ranks stay equal, and all 235 uncapped matcher
ranks are unchanged. The Sheets query is identical in 90 calls across three
rebuilds, versus two baseline results. Whole raw stores remain byte-identical.
Some other questions still have nondeterministic uncapped tails. This is a
bounded reproducibility fix, not universal determinism or floor restoration.

Tests use synthetic identifiers and identical evidence, insertion in both
orders, fifty repeat queries, exact cutoff membership, equivalent timestamps
in different zones, and newer-first precedence. Full suite, restored-source
benchmarks, uncapped repeats and fresh independent disproof remain required.
This is not a proposed restoration of the 53/34 floors: a stable tie can
select either of the previously possible equal candidates. No claim of
unchanged full benchmark payloads or universality follows before measurement.

The first test invocation found a compile-time helper-name typo in the new
recall comparator; it was corrected from factHitKey to existing hitKey before
continuing. No binary containing the new change has been deployed.

Do not combine this with the rejected original-query or relative-reason
variants. All live facts, source-retention rules, marker decisions and running
binaries remain unchanged by this source-only proposal.
