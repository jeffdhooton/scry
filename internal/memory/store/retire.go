package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// EntityRetirementRequest is one reviewed conversion of a non-identity node.
// A first dry run needs only Entity and Why; it returns the complete entity,
// alias, and touching-fact snapshot. The reviewer then supplies exactly one
// replacement for every fact and copies Expected into the apply manifest.
type EntityRetirementRequest struct {
	ID                  string                            `json:"id,omitempty"`
	Entity              string                            `json:"entity"`
	Replacements        []EntityRetirementReplacement     `json:"replacements,omitempty"`
	ReviewedAdjacencies []EntityRetirementAdjacencyReview `json:"reviewed_adjacencies,omitempty"`
	RehomeAliases       []EntityRetirementAliasRehome     `json:"rehome_aliases,omitempty"`
	Expected            EntityRetirementExpected          `json:"expected,omitempty"`
	Why                 string                            `json:"why"`
}

// EntityRetirementAdjacencyReview explicitly authorizes removal of an
// anomalous reverse-index record. Ordinary empty mirrors of reviewed facts
// need no separate entry; stale records and nonempty values always do.
type EntityRetirementAdjacencyReview struct {
	Key            string `json:"key"`
	ExpectedSHA256 string `json:"expected_sha256"`
	Why            string `json:"why"`
}

// EntityRetirementAliasRehome preserves a spelling that legitimately belongs
// to another existing identity. The target must already list the spelling;
// retirement never invents an owner from the name.
type EntityRetirementAliasRehome struct {
	Alias  string `json:"alias"`
	Entity string `json:"entity"`
	Why    string `json:"why"`
}

// EntityRetirementReplacement binds a reviewed replacement fact to the exact
// old key and content fingerprint it replaces. Replacement may turn an edge
// into an attribute or relocate an endpoint, but it may not change the fact's
// text, time, validity, confidence, raw relation, or episode provenance.
type EntityRetirementReplacement struct {
	OldKey         string `json:"old_key"`
	ExpectedSHA256 string `json:"expected_sha256"`
	Replacement    Fact   `json:"replacement"`
	Why            string `json:"why"`
}

type EntityRetirementExpected struct {
	Entities    map[string]string `json:"entities,omitempty"`
	Facts       string            `json:"facts,omitempty"`
	AliasClaims string            `json:"alias_claims,omitempty"`
	Adjacencies string            `json:"adjacencies,omitempty"`
}

// EntityRetirementAdjacencyFingerprint exposes every reverse-index record
// whose encoded source or destination is the retired entity. Stale records
// have no matching canonical fact, but are still reviewed, fingerprinted,
// and removed by the retirement transaction.
type EntityRetirementAdjacencyFingerprint struct {
	Key              string `json:"key"`
	SHA256           string `json:"sha256"`
	CanonicalFactKey string `json:"canonical_fact_key"`
	Stale            bool   `json:"stale"`
	ValueBase64      string `json:"value_base64"`
	ValueBytes       int    `json:"value_bytes"`
	NeedsReview      bool   `json:"needs_review"`
}

type EntityRetirementPreview struct {
	ID                  string                                 `json:"id,omitempty"`
	Entity              Entity                                 `json:"entity_snapshot"`
	Expected            EntityRetirementExpected               `json:"expected"`
	FactFingerprints    []EntityMergeFactFingerprint           `json:"fact_fingerprints"`
	Adjacencies         []EntityRetirementAdjacencyFingerprint `json:"adjacency_fingerprints"`
	ReviewedAdjacencies []EntityRetirementAdjacencyReview      `json:"reviewed_adjacencies,omitempty"`
	Replacements        []EntityRetirementReplacement          `json:"replacements,omitempty"`
	CurrentFacts        int                                    `json:"current_facts"`
	InvalidatedFacts    int                                    `json:"invalidated_facts"`
	AliasClaims         map[string]string                      `json:"alias_claims"`
	RehomeAliases       []EntityRetirementAliasRehome          `json:"rehome_aliases,omitempty"`
	ExternalListings    []string                               `json:"external_listings,omitempty"`
	SelfLoops           []string                               `json:"self_loops,omitempty"`
	FactKeyCollisions   []string                               `json:"fact_key_collisions,omitempty"`
	Problems            []string                               `json:"problems,omitempty"`
	Ready               bool                                   `json:"ready"`
	Applied             bool                                   `json:"applied"`
}

type entityRetirementAnalysis struct {
	preview       EntityRetirementPreview
	entity        Entity
	facts         []Fact
	replacements  []Fact
	claimNorms    map[string]string
	rehomeNorms   map[string]string
	rehomeTargets map[string]Entity
	allFactCount  int
	adjacencyKeys [][]byte
}

func (s *Store) PreviewEntityRetirement(req EntityRetirementRequest) (EntityRetirementPreview, error) {
	var analysis entityRetirementAnalysis
	err := s.db.View(func(txn *badger.Txn) error {
		var err error
		analysis, err = analyzeEntityRetirementTxn(txn, req)
		return err
	})
	return analysis.preview, err
}

func (s *Store) RetireEntity(req EntityRetirementRequest) (EntityRetirementPreview, error) {
	return s.RetireEntityChecked(req, nil)
}

// RetireEntityChecked applies a complete retirement in one Badger
// transaction. The optional postcondition sees a read-your-writes snapshot;
// returning an error aborts every fact, entity, and alias mutation.
func (s *Store) RetireEntityChecked(req EntityRetirementRequest, postcondition func([]Entity, []Fact) error) (EntityRetirementPreview, error) {
	previews, err := s.RetireEntitiesChecked([]EntityRetirementRequest{req}, postcondition)
	if len(previews) == 0 {
		return EntityRetirementPreview{}, err
	}
	return previews[0], err
}

// RetireEntitiesChecked commits a reviewed manifest as one transaction. A
// drift or failed postcondition in any group aborts every group.
func (s *Store) RetireEntitiesChecked(reqs []EntityRetirementRequest, postcondition func([]Entity, []Fact) error) ([]EntityRetirementPreview, error) {
	s.maintenanceMu.RLock()
	defer s.maintenanceMu.RUnlock()
	return s.retireEntitiesCheckedUnlocked(reqs, postcondition)
}

type durableBackupWriter interface {
	io.Writer
	Sync() error
	Close() error
}

// BackupAndRetireEntities holds the store's exclusive maintenance lock from
// the backup snapshot through the manifest transaction. The backup is synced
// and closed before any write. Ordinary writers cannot land a change between
// that verified rollback point and apply.
func (s *Store) BackupAndRetireEntities(w durableBackupWriter, reqs []EntityRetirementRequest) (uint64, []EntityRetirementPreview, error) {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	n, err := s.backupUnlocked(w)
	if err != nil {
		_ = w.Close()
		return n, nil, err
	}
	if err := w.Sync(); err != nil {
		_ = w.Close()
		return n, nil, fmt.Errorf("sync retirement backup: %w", err)
	}
	if err := w.Close(); err != nil {
		return n, nil, fmt.Errorf("close retirement backup: %w", err)
	}
	previews, err := s.retireEntitiesCheckedUnlocked(reqs, nil)
	return n, previews, err
}

func (s *Store) retireEntitiesCheckedUnlocked(reqs []EntityRetirementRequest, postcondition func([]Entity, []Fact) error) ([]EntityRetirementPreview, error) {
	if len(reqs) == 0 {
		return nil, errors.New("memory: retirement manifest is empty")
	}
	analyses := make([]entityRetirementAnalysis, len(reqs))
	err := s.db.Update(func(txn *badger.Txn) error {
		for i, req := range reqs {
			var err error
			analyses[i], err = analyzeEntityRetirementTxn(txn, req)
			if err != nil {
				return err
			}
			if err := verifyRetirementExpected(req.Expected, analyses[i].preview.Expected); err != nil {
				return err
			}
			if !analyses[i].preview.Ready {
				return fmt.Errorf("memory: retirement %s is not ready: %s", retirementLabel(req), strings.Join(analyses[i].preview.Problems, "; "))
			}
			if err := applyEntityRetirementTxn(txn, req, analyses[i]); err != nil {
				return err
			}
		}
		if postcondition != nil {
			postEntities, err := entitiesTxn(txn)
			if err != nil {
				return err
			}
			postFacts, err := factsTxn(txn)
			if err != nil {
				return err
			}
			if err := postcondition(postEntities, postFacts); err != nil {
				return fmt.Errorf("memory: retirement postcondition: %w", err)
			}
		}
		return nil
	})
	previews := make([]EntityRetirementPreview, len(analyses))
	for i := range analyses {
		previews[i] = analyses[i].preview
	}
	if err != nil {
		return previews, err
	}
	for i := range analyses {
		previews[i].Applied = true
		for j, old := range analyses[i].facts {
			s.notify(Event{Kind: "fact", Op: "delete", Fact: old})
			s.notify(Event{Kind: "fact", Op: "put", Fact: analyses[i].replacements[j]})
		}
		s.notify(Event{Kind: "entity", Op: "delete", Slug: reqs[i].Entity})
	}
	return previews, nil
}

func applyEntityRetirementTxn(txn *badger.Txn, req EntityRetirementRequest, analysis entityRetirementAnalysis) error {
	for _, fact := range analysis.facts {
		if err := txn.Delete(factKey(fact.Src, fact.Relation, fact.KeyDst(), fact.ValidFrom)); err != nil {
			return err
		}
		if fact.Dst != "" {
			if err := txn.Delete(adjKey(fact.Dst, fact.Src, fact.Relation, fact.ValidFrom)); err != nil {
				return err
			}
		}
	}
	// Remove every reviewed reverse-index record before writing replacement
	// mirrors. This includes ghosts at a replacement's future adjacency key;
	// deleting after the writes would erase the newly canonical mirror.
	for _, key := range analysis.adjacencyKeys {
		if err := txn.Delete(key); err != nil {
			return err
		}
	}
	for _, fact := range analysis.replacements {
		encoded, err := json.Marshal(fact)
		if err != nil {
			return err
		}
		if err := txn.Set(factKey(fact.Src, fact.Relation, fact.KeyDst(), fact.ValidFrom), encoded); err != nil {
			return err
		}
		if fact.Dst != "" {
			if err := txn.Set(adjKey(fact.Dst, fact.Src, fact.Relation, fact.ValidFrom), nil); err != nil {
				return err
			}
		}
	}
	for norm := range analysis.claimNorms {
		// Every relevant claim was fingerprinted and reviewed. Clear it
		// regardless of its legacy owner, then restore only explicit rehomes;
		// preserving a stale wrong-owner claim would leave an index ghost.
		if err := txn.Delete([]byte(prefixAlias + norm)); err != nil {
			return err
		}
	}
	for norm, target := range analysis.rehomeNorms {
		if err := txn.Set([]byte(prefixAlias+norm), []byte(target)); err != nil {
			return err
		}
	}
	if err := txn.Delete([]byte(prefixEntity + req.Entity)); err != nil {
		return err
	}

	if _, err := txn.Get([]byte(prefixEntity + req.Entity)); !errors.Is(err, badger.ErrKeyNotFound) {
		return fmt.Errorf("memory: retirement postcondition: entity %s still exists", req.Entity)
	}
	for norm := range analysis.claimNorms {
		owner, found, err := aliasOwnerTxn(txn, norm)
		if err != nil {
			return err
		}
		if _, rehomed := analysis.rehomeNorms[norm]; !rehomed && found {
			return fmt.Errorf("memory: retirement postcondition: dropped alias %q still resolves to %q", norm, owner)
		}
	}
	for norm, target := range analysis.rehomeNorms {
		owner, found, err := aliasOwnerTxn(txn, norm)
		if err != nil || !found || owner != target {
			return fmt.Errorf("memory: retirement postcondition: rehomed alias %q resolves to %q, found=%v: %w", norm, owner, found, err)
		}
	}
	postFacts, err := factsTxn(txn)
	if err != nil {
		return err
	}
	if len(postFacts) != analysis.allFactCount {
		return errors.New("memory: retirement postcondition: fact count changed")
	}
	for _, fact := range postFacts {
		if fact.Src == req.Entity || fact.Dst == req.Entity {
			return fmt.Errorf("memory: retirement postcondition: fact still references %s", req.Entity)
		}
	}
	remainingAdjacencies, err := adjacencyReferencesTxn(txn, req.Entity, nil, nil)
	if err != nil {
		return err
	}
	if len(remainingAdjacencies) != 0 {
		return fmt.Errorf("memory: retirement postcondition: adjacency still references %s: %s", req.Entity, remainingAdjacencies[0].Key)
	}
	for _, want := range analysis.replacements {
		item, err := txn.Get(factKey(want.Src, want.Relation, want.KeyDst(), want.ValidFrom))
		if err != nil {
			return fmt.Errorf("memory: retirement postcondition: replacement fact missing: %w", err)
		}
		var got Fact
		if err := item.Value(func(value []byte) error { return json.Unmarshal(value, &got) }); err != nil {
			return err
		}
		if !reflect.DeepEqual(got, want) {
			return errors.New("memory: retirement postcondition: replacement fact changed")
		}
		if want.Dst != "" {
			adjacencyItem, err := txn.Get(adjKey(want.Dst, want.Src, want.Relation, want.ValidFrom))
			if err != nil {
				return fmt.Errorf("memory: retirement postcondition: replacement adjacency missing: %w", err)
			}
			if err := adjacencyItem.Value(func(value []byte) error {
				if len(value) != 0 {
					return errors.New("memory: retirement postcondition: replacement adjacency is nonempty")
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func analyzeEntityRetirementTxn(txn *badger.Txn, req EntityRetirementRequest) (entityRetirementAnalysis, error) {
	a := entityRetirementAnalysis{claimNorms: map[string]string{}, rehomeNorms: map[string]string{}, rehomeTargets: map[string]Entity{}}
	a.preview = EntityRetirementPreview{ID: req.ID, AliasClaims: map[string]string{}, Replacements: append([]EntityRetirementReplacement(nil), req.Replacements...), ReviewedAdjacencies: append([]EntityRetirementAdjacencyReview(nil), req.ReviewedAdjacencies...), RehomeAliases: append([]EntityRetirementAliasRehome(nil), req.RehomeAliases...)}
	problem := func(message string) { a.preview.Problems = append(a.preview.Problems, message) }
	if strings.TrimSpace(req.Entity) == "" {
		return a, errors.New("memory: retirement entity is required")
	}
	if strings.TrimSpace(req.Why) == "" {
		problem("retirement requires a reason")
	}
	entity, err := getEntityTxn(txn, req.Entity)
	if errors.Is(err, ErrNotFound) {
		problem("entity does not exist: " + req.Entity)
		a.preview.Ready = false
		return a, nil
	}
	if err != nil {
		return a, err
	}
	a.entity = entity
	a.preview.Entity = entity

	entities, err := entitiesTxn(txn)
	if err != nil {
		return a, err
	}
	entitiesBySlug := make(map[string]Entity, len(entities))
	for _, candidate := range entities {
		entitiesBySlug[candidate.Slug] = candidate
	}
	claims, err := aliasClaimsTxn(txn)
	if err != nil {
		return a, err
	}
	for _, spelling := range append([]string{entity.Slug, entity.Name}, entity.Aliases...) {
		if norm := Normalize(spelling); norm != "" {
			a.claimNorms[norm] = spelling
		}
	}
	for norm, owner := range claims {
		if owner == req.Entity {
			a.claimNorms[norm] = norm
		}
	}
	outsideByNorm := map[string][]string{}
	for norm := range a.claimNorms {
		if owner, found := claims[norm]; found {
			a.preview.AliasClaims[norm] = owner
		}
		for _, candidate := range entities {
			if candidate.Slug != req.Entity && entityListsNormalized(candidate, norm) {
				a.preview.ExternalListings = append(a.preview.ExternalListings, norm+" listed by "+candidate.Slug)
				outsideByNorm[norm] = append(outsideByNorm[norm], candidate.Slug)
			}
		}
	}
	seenRehomes := map[string]bool{}
	for _, rehome := range req.RehomeAliases {
		norm := Normalize(rehome.Alias)
		if norm == "" || strings.TrimSpace(rehome.Entity) == "" || strings.TrimSpace(rehome.Why) == "" {
			problem("every alias rehome requires alias, entity, and reason")
			continue
		}
		if seenRehomes[norm] {
			problem("alias rehome is repeated: " + rehome.Alias)
			continue
		}
		seenRehomes[norm] = true
		if _, relevant := a.claimNorms[norm]; !relevant {
			problem("alias rehome is not a retired spelling: " + rehome.Alias)
			continue
		}
		target, found := entitiesBySlug[rehome.Entity]
		if !found || rehome.Entity == req.Entity || !entityListsNormalized(target, norm) {
			problem(fmt.Sprintf("alias rehome target %q does not exist or list %q", rehome.Entity, rehome.Alias))
			continue
		}
		a.rehomeNorms[norm] = rehome.Entity
		a.rehomeTargets[rehome.Entity] = target
	}
	for norm, listings := range outsideByNorm {
		target := a.rehomeNorms[norm]
		for _, listing := range listings {
			if target == "" || listing != target {
				problem(fmt.Sprintf("retired spelling %q remains listed by %s; rehome it there or review it with unalias first", norm, listing))
			}
		}
	}

	facts, err := factsTxn(txn)
	if err != nil {
		return a, err
	}
	a.allFactCount = len(facts)
	allFactKeys := make(map[string]bool, len(facts))
	allFactsByKey := make(map[string]Fact, len(facts))
	oldByKey := map[string]Fact{}
	for _, fact := range facts {
		key := string(factKey(fact.Src, fact.Relation, fact.KeyDst(), fact.ValidFrom))
		allFactKeys[key] = true
		allFactsByKey[key] = fact
		if fact.Src != req.Entity && fact.Dst != req.Entity {
			continue
		}
		if fact.Dst != "" && fact.Value != "" {
			problem("touching fact is malformed with both dst and value: " + key)
		}
		a.facts = append(a.facts, fact)
		oldByKey[key] = fact
		delete(allFactKeys, key)
		a.preview.FactFingerprints = append(a.preview.FactFingerprints, EntityMergeFactFingerprint{
			Key: key, SHA256: hashJSON(fact), Invalidated: fact.InvalidAt != nil, Fact: fact.Fact, Snapshot: fact,
		})
		if fact.InvalidAt == nil {
			a.preview.CurrentFacts++
		} else {
			a.preview.InvalidatedFacts++
		}
	}
	seenOld := map[string]bool{}
	targetKeys := map[string]string{}
	inputByKey := map[string]EntityRetirementReplacement{}
	for _, replacement := range req.Replacements {
		old, found := oldByKey[replacement.OldKey]
		if !found {
			problem("replacement names unknown old_key: " + replacement.OldKey)
			continue
		}
		if seenOld[replacement.OldKey] {
			problem("replacement repeats old_key: " + replacement.OldKey)
			continue
		}
		seenOld[replacement.OldKey] = true
		inputByKey[replacement.OldKey] = replacement
		if replacement.ExpectedSHA256 == "" || replacement.ExpectedSHA256 != hashJSON(old) {
			problem("replacement old fact fingerprint mismatch: " + replacement.OldKey)
		}
		if strings.TrimSpace(replacement.Why) == "" {
			problem("replacement requires a reason: " + replacement.OldKey)
		}
		updated := replacement.Replacement
		if !sameRetirementFactPayload(old, updated) {
			problem("replacement changes fact text, time, validity, confidence, relation, or provenance: " + replacement.OldKey)
		}
		if reason := invalidRetirementFactShape(req.Entity, old, updated); reason != "" {
			problem(reason + ": " + replacement.OldKey)
		}
		if updated.Src == req.Entity || updated.Dst == req.Entity {
			problem("replacement still references retired entity: " + replacement.OldKey)
		}
		if updated.Src == "" || (updated.Dst == "" && updated.Value == "") || (updated.Dst != "" && updated.Value != "") {
			problem("replacement must have a source and exactly one of dst/value: " + replacement.OldKey)
		}
		for _, endpoint := range []string{updated.Src, updated.Dst} {
			if endpoint == "" {
				continue
			}
			if _, found := entitiesBySlug[endpoint]; !found || endpoint == req.Entity {
				problem("replacement has missing or retired endpoint " + endpoint + ": " + replacement.OldKey)
			}
		}
		if updated.Dst != "" && updated.Src == updated.Dst {
			a.preview.SelfLoops = append(a.preview.SelfLoops, replacement.OldKey)
			problem("replacement creates a self-loop: " + replacement.OldKey)
		}
		newKey := string(factKey(updated.Src, updated.Relation, updated.KeyDst(), updated.ValidFrom))
		if allFactKeys[newKey] {
			a.preview.FactKeyCollisions = append(a.preview.FactKeyCollisions, replacement.OldKey+" -> "+newKey)
			problem("replacement collides with an untouched fact: " + replacement.OldKey)
		}
		if prior, found := targetKeys[newKey]; found {
			a.preview.FactKeyCollisions = append(a.preview.FactKeyCollisions, prior+" + "+replacement.OldKey+" -> "+newKey)
			problem("replacements create duplicate fact keys")
		}
		targetKeys[newKey] = replacement.OldKey
	}
	for _, fingerprint := range a.preview.FactFingerprints {
		if !seenOld[fingerprint.Key] {
			problem("touching fact has no reviewed replacement: " + fingerprint.Key)
		}
	}
	for _, fingerprint := range a.preview.FactFingerprints {
		if replacement, found := inputByKey[fingerprint.Key]; found {
			a.replacements = append(a.replacements, replacement.Replacement)
		}
	}
	replacementAdjacencyKeys := map[string]bool{}
	for _, replacement := range a.replacements {
		if replacement.Dst != "" {
			replacementAdjacencyKeys[string(adjKey(replacement.Dst, replacement.Src, replacement.Relation, replacement.ValidFrom))] = true
		}
	}
	adjacencies, err := adjacencyReferencesTxn(txn, req.Entity, allFactsByKey, replacementAdjacencyKeys)
	if err != nil {
		return a, err
	}
	a.preview.Adjacencies = adjacencies
	for _, adjacency := range adjacencies {
		a.adjacencyKeys = append(a.adjacencyKeys, []byte(adjacency.Key))
	}
	adjacencyByKey := make(map[string]EntityRetirementAdjacencyFingerprint, len(adjacencies))
	for _, adjacency := range adjacencies {
		adjacencyByKey[adjacency.Key] = adjacency
	}
	seenAdjacencyReviews := map[string]bool{}
	for _, review := range req.ReviewedAdjacencies {
		adjacency, found := adjacencyByKey[review.Key]
		if !found {
			problem("adjacency review names unknown key: " + review.Key)
			continue
		}
		if seenAdjacencyReviews[review.Key] {
			problem("adjacency review repeats key: " + review.Key)
			continue
		}
		seenAdjacencyReviews[review.Key] = true
		if review.ExpectedSHA256 == "" || review.ExpectedSHA256 != adjacency.SHA256 {
			problem("adjacency review fingerprint mismatch: " + review.Key)
		}
		if strings.TrimSpace(review.Why) == "" {
			problem("adjacency review requires a reason: " + review.Key)
		}
		if !adjacency.NeedsReview {
			problem("adjacency review is unnecessary for canonical empty mirror: " + review.Key)
		}
	}
	for _, adjacency := range adjacencies {
		if adjacency.NeedsReview && !seenAdjacencyReviews[adjacency.Key] {
			problem("anomalous adjacency has no explicit review: " + adjacency.Key)
		}
	}
	expectedEntities := map[string]string{req.Entity: hashJSON(entity)}
	for slug, target := range a.rehomeTargets {
		expectedEntities[slug] = hashJSON(target)
	}
	for _, replacement := range a.replacements {
		for _, endpoint := range []string{replacement.Src, replacement.Dst} {
			if endpoint != "" && endpoint != req.Entity {
				if target, found := entitiesBySlug[endpoint]; found {
					expectedEntities[endpoint] = hashJSON(target)
				}
			}
		}
	}
	a.preview.Expected = EntityRetirementExpected{
		Entities: expectedEntities, Facts: hashJSON(a.facts), AliasClaims: hashAliasSubset(a.claimNorms, claims), Adjacencies: hashJSON(adjacencies),
	}
	sort.Strings(a.preview.ExternalListings)
	sort.Strings(a.preview.Problems)
	a.preview.Ready = len(a.preview.Problems) == 0
	return a, nil
}

func adjacencyReferencesTxn(txn *badger.Txn, retired string, factsByKey map[string]Fact, replacementKeys map[string]bool) ([]EntityRetirementAdjacencyFingerprint, error) {
	var fingerprints []EntityRetirementAdjacencyFingerprint
	prefix := []byte(prefixAdj)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	it := txn.NewIterator(opts)
	defer it.Close()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		key := item.KeyCopy(nil)
		dst, src, relation, validFrom, valid := parseAdjacencyKey(key)
		if dst != retired && src != retired && !replacementKeys[string(key)] {
			continue
		}
		var value []byte
		if err := item.Value(func(raw []byte) error {
			value = append(value, raw...)
			return nil
		}); err != nil {
			return nil, err
		}
		canonicalKey := ""
		corresponds := false
		if valid {
			canonicalKey = string(factKey(src, relation, dst, validFrom))
			fact, found := factsByKey[canonicalKey]
			corresponds = found && fact.Src == src && fact.Dst == dst && fact.Relation == relation && fact.ValidFrom.Equal(validFrom)
		}
		fingerprints = append(fingerprints, EntityRetirementAdjacencyFingerprint{
			Key: string(key), SHA256: hashJSON(struct {
				Key   string `json:"key"`
				Value []byte `json:"value"`
			}{Key: string(key), Value: value}), CanonicalFactKey: canonicalKey, Stale: !corresponds,
			ValueBase64: base64.StdEncoding.EncodeToString(value), ValueBytes: len(value), NeedsReview: !corresponds || len(value) != 0,
		})
	}
	if fingerprints == nil {
		fingerprints = []EntityRetirementAdjacencyFingerprint{}
	}
	return fingerprints, nil
}

func parseAdjacencyKey(key []byte) (dst, src, relation string, validFrom time.Time, ok bool) {
	rest := strings.TrimPrefix(string(key), prefixAdj)
	parts := strings.SplitN(rest, ":", 4)
	if len(parts) > 0 {
		dst = parts[0]
	}
	if len(parts) > 1 {
		src = parts[1]
	}
	if len(parts) > 2 {
		relation = parts[2]
	}
	if len(parts) != 4 || dst == "" || src == "" || relation == "" {
		return dst, src, relation, time.Time{}, false
	}
	nano, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return dst, src, relation, time.Time{}, false
	}
	return parts[0], parts[1], parts[2], time.Unix(0, nano).UTC(), true
}

func sameRetirementFactPayload(old, updated Fact) bool {
	return old.Relation == updated.Relation && old.RawRelation == updated.RawRelation &&
		old.Fact == updated.Fact && old.ValidFrom.Equal(updated.ValidFrom) &&
		reflect.DeepEqual(old.InvalidAt, updated.InvalidAt) && old.Confidence == updated.Confidence &&
		reflect.DeepEqual(old.Episodes, updated.Episodes) &&
		(old.Value == "" || old.Value == updated.Value)
}

// invalidRetirementFactShape constrains a replacement to the smallest change
// that can remove the retired endpoint. A non-retired endpoint is immutable.
// Existing attributes keep their literal and can only move their retired
// source. An edge can become an attribute only when its retired destination
// is the value being de-noded; the source then remains unchanged.
func invalidRetirementFactShape(retired string, old, updated Fact) string {
	if old.Dst == "" {
		if updated.Dst != "" {
			return "replacement turns an attribute into an edge"
		}
		if updated.Value != old.Value {
			return "replacement changes an existing attribute value"
		}
		if old.Src != retired || updated.Src == old.Src {
			return "attribute replacement does not relocate its retired source"
		}
		return ""
	}

	if updated.Dst == "" {
		if old.Dst != retired || old.Src == retired || updated.Src != old.Src || updated.Value == "" {
			return "edge-to-attribute replacement must convert only a retired destination"
		}
		return ""
	}

	if old.Src != retired && updated.Src != old.Src {
		return "replacement changes a non-retired source"
	}
	if old.Dst != retired && updated.Dst != old.Dst {
		return "replacement changes a non-retired destination"
	}
	if old.Src == retired && updated.Src == old.Src {
		return "replacement does not relocate its retired source"
	}
	if old.Dst == retired && updated.Dst == old.Dst {
		return "replacement does not relocate its retired destination"
	}
	if updated.Value != "" {
		return "edge replacement carries an attribute value"
	}
	return ""
}

func verifyRetirementExpected(got, want EntityRetirementExpected) error {
	if got.Facts == "" || got.AliasClaims == "" || got.Adjacencies == "" || len(got.Entities) == 0 {
		return errors.New("memory: retirement expected entity, fact, alias, and adjacency fingerprints are required for apply")
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("memory: retirement snapshot changed: got %+v, want %+v", want, got)
	}
	return nil
}

func retirementLabel(req EntityRetirementRequest) string {
	if req.ID != "" {
		return req.ID
	}
	return req.Entity
}
