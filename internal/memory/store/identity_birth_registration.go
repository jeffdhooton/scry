package store

// PRIVATE, UNCALLED candidate/history accounting. No support, routing, B votes,
// durable observations, current-result or production admission policy.

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

var errBirthRegistration = errors.New("memory: invalid candidate birth registration")

type registeredObservation struct {
	Key  string
	Raw  []byte
	Role string
}

type registeredBirth struct {
	registry *birthRegistry
	birth    identityBirth
	first    registeredObservation
	position uint64
	mentions []registeredObservation
}

type birthRegistrationDecision struct {
	Kind        string
	Handle      *registeredBirth
	Observation registeredObservation
}

type birthCandidateReport struct {
	Birth        identityBirth
	First        registeredObservation
	Position     uint64
	Mentions     []registeredObservation
	Created      bool
	FinalPresent bool
}

type birthRegistryReport struct {
	RevisionKey  string
	RevisionRaw  []byte
	Candidates   []birthCandidateReport
	Dispositions []birthRegistrationDecision
	Identities   identityMutationReport
	Facts        identityFactLedgerReport
}

type birthRegistry struct {
	st           *Store
	owner        *identityAdmissionOwner
	revision     identityInputRevision
	revisionKey  string
	revisionRaw  []byte
	identities   *identityMutationLedger
	facts        *identityFactLedger
	bySlug       map[string]*registeredBirth
	ordered      []*registeredBirth
	dispositions []birthRegistrationDecision
	frozen       bool
}

func birthRegistrationFailure(st *Store, err error) error {
	safe := error(errBirthRegistration)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errBirthRegistration, kind)
			break
		}
	}
	if st != nil && st.txn != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

// This entry fixes initialization and verification; no caller finalizer is
// accepted. It still certifies only the finite mechanical contract above.
func runBirthRegistration(st *Store, revisionKey string, revisionRaw []byte, body func(*birthRegistry) error) (birthRegistryReport, error) {
	if st == nil {
		return birthRegistryReport{}, errBirthRegistration
	}
	if st.txn != nil {
		birthRegistrationFailure(st, errBirthRegistration)
		return birthRegistryReport{}, st.admissionFailure
	}
	if body == nil {
		return birthRegistryReport{}, errBirthRegistration
	}
	// Own bytes before waiting; callbacks cannot alter the submitted revision.
	revisionRaw = bytes.Clone(revisionRaw)
	var registry *birthRegistry
	var report birthRegistryReport
	err := runSerializedIdentityAdmission(st, func(tx *Store) error {
		// Decode with the embedded ID only to establish the expected complete
		// canonical key/body. No EP row or ownership is implied by this input.
		var err error
		registry, err = newBirthRegistry(tx, revisionKey, revisionRaw)
		if err != nil {
			return err
		}
		return body(registry)
	}, func(*Store) error {
		var err error
		report, err = registry.freeze()
		return err
	})
	if err != nil {
		return birthRegistryReport{}, err
	}
	return report, nil
}

func newBirthRegistry(st *Store, key string, raw []byte) (*birthRegistry, error) {
	if st == nil || st.admissionOwner == nil || st.admissionOwner.phase != admissionBody || st.admissionFailure != nil {
		return nil, birthRegistrationFailure(st, errBirthRegistration)
	}
	// The complete decoder below checks key, all fields and canonical bytes.
	r, err := decodeBirthRevision(key, raw)
	if err != nil {
		return nil, birthRegistrationFailure(st, err)
	}
	identities, err := beginIdentityMutationLedger(st)
	if err != nil {
		return nil, birthRegistrationFailure(st, err)
	}
	facts, err := beginIdentityFactLedger(st)
	if err != nil {
		return nil, birthRegistrationFailure(st, err)
	}
	return &birthRegistry{st: st, owner: st.admissionOwner, revision: r, revisionKey: key, revisionRaw: bytes.Clone(raw), identities: identities, facts: facts, bySlug: map[string]*registeredBirth{}, ordered: []*registeredBirth{}, dispositions: []birthRegistrationDecision{}}, nil
}

func decodeBirthRevision(key string, raw []byte) (identityInputRevision, error) {
	var r identityInputRevision
	if json.Unmarshal(raw, &r) != nil {
		return identityInputRevision{}, errBirthRegistration
	}
	return decodeIdentityInput(key, raw, r.EpisodeID)
}

func (r *birthRegistry) phase(want int) error {
	if r == nil {
		return errBirthRegistration
	}
	if r.st == nil || r.owner == nil || r.st.admissionOwner != r.owner || r.owner.phase != want || r.st.admissionFailure != nil || r.frozen || r.bySlug == nil || r.identities == nil || r.facts == nil || r.identities.st != r.st || r.facts.st != r.st || r.identities.owner != r.owner || r.facts.owner != r.owner {
		return birthRegistrationFailure(r.st, errBirthRegistration)
	}
	if err := generationTransaction(r.st); err != nil {
		return birthRegistrationFailure(r.st, err)
	}
	return nil
}

func cloneRegisteredObservation(o registeredObservation) registeredObservation {
	o.Raw = bytes.Clone(o.Raw)
	return o
}

func sameRegisteredObservation(a, b registeredObservation) bool {
	return a.Key == b.Key && a.Role == b.Role && bytes.Equal(a.Raw, b.Raw)
}

func (r *birthRegistry) observation(key string, raw []byte, role string) (identityObservation, registeredObservation, error) {
	if err := matchRevisionObservation(r.revisionKey, r.revisionRaw, key, raw, r.revision.EpisodeID); err != nil {
		return identityObservation{}, registeredObservation{}, birthRegistrationFailure(r.st, err)
	}
	o, err := decodeIdentityObservation(key, raw, r.revision.EpisodeID)
	if err != nil {
		return identityObservation{}, registeredObservation{}, birthRegistrationFailure(r.st, err)
	}
	valid := false
	switch role {
	case "declaration":
		valid = o.Origin == "declaration"
	case "primary-src":
		valid = o.Origin == "endpoint" && o.Side == "src"
	case "primary-dst":
		valid = o.Origin == "endpoint" && o.Side == "dst"
	case "supersedes-src", "supersedes-dst":
		valid = o.Origin == "endpoint" && o.Fact.Supersedes != nil
	}
	if !valid {
		return identityObservation{}, registeredObservation{}, birthRegistrationFailure(r.st, errBirthRegistration)
	}
	return o, registeredObservation{Key: key, Raw: bytes.Clone(raw), Role: role}, nil
}

func (r *birthRegistry) normalDecision(kind string, observation registeredObservation) birthRegistrationDecision {
	decision := birthRegistrationDecision{Kind: kind, Observation: cloneRegisteredObservation(observation)}
	for _, old := range r.dispositions {
		if old.Kind == kind && sameRegisteredObservation(old.Observation, observation) {
			return decision
		}
	}
	r.dispositions = append(r.dispositions, birthRegistrationDecision{Kind: kind, Observation: cloneRegisteredObservation(observation)})
	return decision
}

func (r *birthRegistry) register(key string, raw []byte) (birthRegistrationDecision, error) {
	if err := r.phase(admissionBody); err != nil {
		return birthRegistrationDecision{}, err
	}
	parsed, err := decodeIdentityObservation(key, raw, r.revision.EpisodeID)
	if err != nil {
		return birthRegistrationDecision{}, birthRegistrationFailure(r.st, err)
	}
	role := "declaration"
	if parsed.Origin == "endpoint" {
		role = "primary-" + parsed.Side
	}
	o, observed, err := r.observation(key, raw, role)
	if err != nil {
		return birthRegistrationDecision{}, err
	}
	name := ""
	if o.Declaration != nil {
		name = o.Declaration.Name
	} else if o.Side == "src" {
		name = o.Fact.Src
	} else {
		name = o.Fact.Dst
	}
	record, _, err := identityBirthRecord(identityBirth{EpisodeID: r.revision.EpisodeID, Occurrence: o.Ordinal, Slug: Slugify(name), Name: name, Origin: o.Origin, CreatedAt: r.revision.OccurredAt})
	if err != nil {
		return birthRegistrationDecision{}, birthRegistrationFailure(r.st, err)
	}
	b := record.Birth
	if existing := r.bySlug[b.Slug]; existing != nil {
		if !sameRegisteredObservation(existing.first, observed) {
			return birthRegistrationDecision{}, birthRegistrationFailure(r.st, errBirthRegistration)
		}
		return birthRegistrationDecision{Kind: "registered", Handle: existing, Observation: cloneRegisteredObservation(observed)}, nil
	}
	if _, exists := r.identities.baseline.Entities[b.Slug]; exists {
		return r.normalDecision("existing-not-new", observed), nil
	}
	for _, entry := range r.identities.writer.entries {
		if entry.Actor == b.Slug {
			return birthRegistrationDecision{}, birthRegistrationFailure(r.st, errBirthRegistration)
		}
	}
	keys := []string{prefixEntity + b.Slug, identityGenerationPrefix + b.Slug, "il:" + b.Slug, "il-consumed:" + b.Slug, prefixRetiredSlug + b.Slug, prefixRetired + Normalize(b.Name), prefixRetired + Normalize(b.Slug)}
	for _, k := range keys {
		if _, exists := r.identities.baseline.Rows[k]; exists {
			return birthRegistrationDecision{}, birthRegistrationFailure(r.st, errBirthRegistration)
		}
		_, exists, err := generationRead(r.st, []byte(k))
		if err != nil || exists {
			return birthRegistrationDecision{}, birthRegistrationFailure(r.st, err)
		}
	}
	refs := r.facts.baseline.References[b.Slug]
	if refs.Current != 0 || refs.Historical != 0 {
		return r.normalDecision("deferred-preexisting-reference", observed), nil
	}
	h := &registeredBirth{registry: r, birth: b, first: cloneRegisteredObservation(observed), position: uint64(len(r.identities.writer.entries)), mentions: []registeredObservation{}}
	r.bySlug[b.Slug], r.ordered = h, append(r.ordered, h)
	return birthRegistrationDecision{Kind: "registered", Handle: h, Observation: cloneRegisteredObservation(observed)}, nil
}

func (r *birthRegistry) link(h *registeredBirth, key string, raw []byte, role string) error {
	if err := r.phase(admissionBody); err != nil {
		return err
	}
	if h == nil || h.registry != r || r.bySlug[h.birth.Slug] != h {
		return birthRegistrationFailure(r.st, errBirthRegistration)
	}
	_, observed, err := r.observation(key, raw, role)
	if err != nil {
		return err
	}
	if sameRegisteredObservation(h.first, observed) {
		return nil
	}
	for _, old := range h.mentions {
		if sameRegisteredObservation(old, observed) {
			return nil
		}
	}
	h.mentions = append(h.mentions, cloneRegisteredObservation(observed))
	return nil
}

func (r *birthRegistry) identityActor(actor string) error {
	if err := r.phase(admissionBody); err != nil {
		return err
	}
	if _, existing := r.identities.baseline.Entities[actor]; !existing && r.bySlug[actor] == nil {
		return birthRegistrationFailure(r.st, errBirthRegistration)
	}
	return nil
}

func (r *birthRegistry) putIdentity(actor string, key, raw []byte) error {
	if err := r.identityActor(actor); err != nil {
		return err
	}
	if err := r.identities.put(actor, key, raw); err != nil {
		return birthRegistrationFailure(r.st, err)
	}
	return nil
}

func (r *birthRegistry) deleteIdentity(actor string, key []byte) error {
	if err := r.identityActor(actor); err != nil {
		return err
	}
	if err := r.identities.delete(actor, key); err != nil {
		return birthRegistrationFailure(r.st, err)
	}
	return nil
}

func (r *birthRegistry) putFact(ordinal int, key, raw []byte) error {
	if err := r.phase(admissionBody); err != nil {
		return err
	}
	if err := r.facts.put(ordinal, key, raw); err != nil {
		return birthRegistrationFailure(r.st, err)
	}
	return nil
}

func (r *birthRegistry) deleteFact(ordinal int, key []byte) error {
	if err := r.phase(admissionBody); err != nil {
		return err
	}
	if err := r.facts.delete(ordinal, key); err != nil {
		return birthRegistrationFailure(r.st, err)
	}
	return nil
}

func (r *birthRegistry) freeze() (birthRegistryReport, error) {
	if err := r.phase(admissionFinalizing); err != nil {
		return birthRegistryReport{}, err
	}
	fail := func(err error) (birthRegistryReport, error) {
		return birthRegistryReport{}, birthRegistrationFailure(r.st, err)
	}
	identities, err := r.identities.verify()
	if err != nil {
		return fail(err)
	}
	facts, err := r.facts.verify()
	if err != nil {
		return fail(err)
	}
	created := map[string]bool{}
	for _, entry := range identities.History.Entries {
		if _, existing := identities.Baseline.Entities[entry.Actor]; existing {
			continue
		}
		h := r.bySlug[entry.Actor]
		if h == nil || h.registry != r || entry.Sequence < h.position {
			return fail(errBirthRegistration)
		}
		if !strings.HasPrefix(string(entry.Key), prefixEntity) {
			continue
		}
		if entry.After.Exists {
			e, err := decodeLegacyEntity(entry.Key, entry.After.Value)
			if err != nil || e.Slug != h.birth.Slug || e.Name != h.birth.Name || !e.CreatedAt.Equal(h.birth.CreatedAt) {
				return fail(errBirthRegistration)
			}
			if !entry.Before.Exists {
				if created[entry.Actor] {
					return fail(errBirthRegistration)
				}
				created[entry.Actor] = true
			}
		}
	}
	out := birthRegistryReport{RevisionKey: r.revisionKey, RevisionRaw: bytes.Clone(r.revisionRaw), Candidates: []birthCandidateReport{}, Dispositions: []birthRegistrationDecision{}, Identities: identities, Facts: facts}
	for _, h := range r.ordered {
		if h == nil || h.registry != r || r.bySlug[h.birth.Slug] != h {
			return fail(errBirthRegistration)
		}
		_, present := identities.Final.Entities[h.birth.Slug]
		candidate := birthCandidateReport{Birth: h.birth, First: cloneRegisteredObservation(h.first), Position: h.position, Mentions: []registeredObservation{}, Created: created[h.birth.Slug], FinalPresent: present}
		for _, mention := range h.mentions {
			candidate.Mentions = append(candidate.Mentions, cloneRegisteredObservation(mention))
		}
		out.Candidates = append(out.Candidates, candidate)
	}
	for _, d := range r.dispositions {
		out.Dispositions = append(out.Dispositions, birthRegistrationDecision{Kind: d.Kind, Observation: cloneRegisteredObservation(d.Observation)})
	}
	r.frozen = true
	return out, nil
}
