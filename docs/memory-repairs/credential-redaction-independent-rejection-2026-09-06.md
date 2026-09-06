# Independent prospective credential-redaction disproof

Verdict: DISPROVED / do not integrate this prototype as reviewed. This verdict concerns the private redaction experiment only. It is not a finding against, and must not delay or get folded into, the separately reviewed fact-key prevention artifact.

## Identity and scope

- Production repository HEAD observed twice: `24eafab441f743da9e27ab1a72debe18c59975c5`. I did not independently interrogate running binaries or the Mini; this identifies the production source checkout, not a fresh binary deployment attestation.
- Candidate: `/tmp/scry-credential-redaction-sep06.BPMYjw`.
- Candidate `internal/memory/distill/redact.go` SHA-256: `e7f7a9d70c4eb5e8a8726109a58ea4c808e9804e5e7aeed39d2bc40e20b1442c`.
- Candidate `internal/memory/distill/credential_assignment_test.go` SHA-256: `edda285d58cf1056b2e46db77bb49e977e0b6c30e6400856eabb5ea178faea40`.
- Reviewed active attached goal, proposal, redactor and tests, distillation/remember/enqueue ingress and extraction handoff. Compared production source against candidate; existing daemon ingress files are identical.
- Fresh independent directory: `/tmp/scry-credential-redaction-disproof-sep06.Qagt65`. `redact.go` is a byte-for-byte candidate copy; `adversarial_test.go` is reviewer-authored.
- All values below are fabricated fixtures. No actual credential, real credential transcript, stored sensitive projection, or associated value-derived identifier was accessed. No provider, queue retry, live/shared/prototype write, deployment, config change, history cleanup, credential action or memory note was performed.

## Proven blockers within the bounded goal

1. Reference and marker alternatives match prefixes, then exempt an incomplete literal. At `redact.go:27` the first matching `$NAME`, `${NAME}` or `[REDACTED]` alternative wins without requiring that the entire supplied token ended. The exemption at line 47 then skips redaction.

   - `PASSWORD=$FAKE91!suffix` stays entirely unchanged.
   - `change the password to $FAKE91!suffix` stays entirely unchanged.
   - `PASSWORD=${FAKE_REF}suffix91!` stays entirely unchanged.
   - `PASSWORD=[REDACTED]FAKE91!` stays entirely unchanged.

   These are long mixed tokens, not the declared all-letter/short/numeric prose limitation. The imperative example is unambiguous password-setting syntax. A real standalone reference such as `PASSWORD=${FAKE_REF}` correctly stays unchanged; checking an incomplete regex capture rather than the entire token is the defect.

2. Unquoted mixed credential values are cut at `]`, `}` and `;`. For explicit env assignments, `]` and `}` are valid literal characters, not reliable value delimiters. In prose, a semicolon can also occur within a supplied token.

   - `PASSWORD=FAKE-91]tail!` becomes `PASSWORD=[REDACTED]]tail!`.
   - `PASSWORD=FAKE-91}tail!` becomes `PASSWORD=[REDACTED]}tail!`.
   - `change the password to FAKE-91]tail!`, `change the password to FAKE-91}tail!` and `change the password to FAKE91;tail!` each stay wholly unchanged because the regex truncates the candidate to fewer than eight bytes before applying the shape test.

   Longer prefixes would redact only the prefix and expose the suffix. Merely retaining these characters as universal stop tokens does not satisfy arbitrary mixed-token prose handling.

3. The new assignment rule is not limited to scalar assignments and can discard useful noncredential context.

   - Input `{"password":null,"project":"Northstar"}` becomes `{"password":[REDACTED]}`: it removes the neighboring project field as well as a nonsecret null. Even a policy of redacting null does not justify losing that field.
   - Input `password:\nproject: Northstar` becomes `password:\n[REDACTED] Northstar`: `\s*` crosses the newline and treats the next field name as the missing password.
   - Input `{"password":{"minLength":12},"project":"Northstar"}` becomes `{"password":[REDACTED]},"project":"Northstar"}`: object configuration is erased, with unmatched closing punctuation.
   - Input `{"password":["required","confirmed"],"project":"Northstar"}` becomes `{"password":[REDACTED]],"project":"Northstar"}`: ordinary validation rules are erased.
   - Input `if password == candidate { allow() }` becomes `if password =[REDACTED] candidate { allow() }`: equality is mistaken for assignment.

4. The prose expression does not establish a password-setting verb, and the mixed-token heuristic rejects ordinary hyphenated advice.

   - `change the password to re-authenticate later` becomes `change the password to [REDACTED] later`.
   - `Use the password to two-factor-enable this account` becomes `Use the password to [REDACTED] this account`.

   The second example does not set or change a password at all. This disproves benign-discussion preservation, not merely universal secret-detection claims.

5. Existing generic redaction can create the exempt prefix and leave a remaining assignment literal exposed. `PASSWORD=ghp_FAKE012345678901234567890!tail91` becomes `PASSWORD=[REDACTED]!tail91`. The existing GitHub-shape rule still does its original job; the new explicit-assignment pass fails to finish redacting the supplied literal because it treats the generated marker as a complete value. This is a composition gap, not a regression claim about the original standalone GitHub regex.

## Additional scope-edge observations

The independent matrix also explores forms which need an explicit scope decision; none is needed to establish the blockers above:

- YAML plain scalar with spaces: `password: FAKE review spaces 91!\nproject: Northstar` retains `review spaces 91!`. The proposal guarantees quoted spaces, so this is a coverage limit unless scalar assignment claims are intended to include YAML plain scalars.
- Smart quotes and Markdown backticks around multiword prose values are not recognized; the complete fabricated value may survive. The implementation currently supports straight single/double quotes only. Document that narrower definition if broader quotes are intentionally excluded.
- A prose credential followed by a sentence period loses the period: `Change the password to FAKE-Review-91!. Preserve this sentence.` becomes `Change the password to [REDACTED] Preserve this sentence.` Punctuation ownership is ambiguous in general; quoted syntax avoids this ambiguity. Do not claim exact unquoted punctuation preservation without defining it.
- All-letter, short and numeric-only unquoted prose values are acknowledged limitations, and were not used as disproof.

## Verification results

- Candidate `go test ./internal/memory/distill ./internal/daemon`: PASS (distill 0.305s; daemon 18.450s).
- Reviewer `go test -v ./...`: FAIL as intended, with 35 explicit literal/context cases: 14 pass and 21 fail. This count includes the scope-edge cases above, so it is not an estimate of production failure rate or a claim that all 21 are release blockers.
- Every one of those 35 outputs was checked again for idempotence; no idempotence counterexample was found.
- Separate `TestIndependentIdempotenceCombinations`: PASS for 320 fabricated combinations of five labels, sixteen literal forms and four suffix contexts. Idempotent partial redaction can still leak text; this does not establish completeness.
- Standalone legacy PEM, Bearer, ghp_ and sk- fixtures all pass. Quoted escaped literals, quoted spaces, straightforward mixed imperative, ordinary straight-quoted JSON values, standalone references and existing redaction markers also pass.

Reproduce in the reviewer directory with `go test -v ./...`; no provider or store is involved. Expected failure output includes only fabricated inputs and results.

## Ingress findings and limits

- `internal/memory/distill/distill.go:149`, `seed.go:55`, and `loom.go:91` redact distilled text before returning episodes. Common conversation sources route through the shared turn chunker.
- `internal/daemon/memory_methods.go:515` redacts manual remember text before deriving its episode ID and calling enqueue.
- `internal/daemon/memory_queue.go:320` redacts `ep.Text` before `PutPending`, providing the common backstop for new enqueue requests.
- `internal/memory/queue/queue.go:388` constructs extraction text directly from pending `p.Text`; there is no newly added worker-side pass. Existing queued items are not repaired by this candidate. No retry/replay was attempted.
- Entity hints are passed separately to the extractor (`queue.go:394`); source refs and other metadata are also outside the `ep.Text` transformation. The proposal should retain its scope of prospective episode/fact text, not all input fields or all storage surfaces.
- Existing source IDs can already be ingested, causing enqueue to return before creating new pending text. This patch cannot erase historical facts, aliases, summaries, transcripts, backups, or old queued data and does not purport to do so.

The appropriate next builder step is another private, bounded syntax revision with the proven counterexamples retained, followed by fresh independent disproof. This review neither implements a fix nor authorizes historical remediation or changes to the separate fact-key deployment.
