package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

// EntityMergeRequest is one reviewed, atomic identity consolidation. A dry
// run may omit Metadata and Expected to obtain a proposed survivor plus exact
// fingerprints. Apply requires the reviewer to copy both back into the
// manifest, making metadata and store state explicit rather than inferred.
type EntityMergeRequest struct {
	ID          string                 `json:"id,omitempty"`
	Survivor    string                 `json:"survivor"`
	Retire      []string               `json:"retire"`
	DropAliases []EntityMergeAliasDrop `json:"drop_aliases,omitempty"`
	Metadata    *Entity                `json:"metadata,omitempty"`
	Expected    EntityMergeExpected    `json:"expected,omitempty"`
	Why         string                 `json:"why,omitempty"`
}

// EntityMergeAliasDrop is an explicit reviewed exception to the default rule
// that every old alias transfers. Entity names and slugs cannot be dropped.
type EntityMergeAliasDrop struct {
	Alias    string `json:"alias"`
	RehomeTo string `json:"rehome_to,omitempty"`
	Why      string `json:"why"`
}

// EntityMergeExpected pins every mutable input the reviewer inspected.
type EntityMergeExpected struct {
	Entities    map[string]string `json:"entities,omitempty"`     // slug -> SHA-256 of entity JSON
	Facts       string            `json:"facts,omitempty"`        // SHA-256 of all touching facts
	AliasClaims string            `json:"alias_claims,omitempty"` // SHA-256 of relevant normalized claims
}

// EntityMergeFactFingerprint is one fact included in the group snapshot.
type EntityMergeFactFingerprint struct {
	Key         string `json:"key"`
	SHA256      string `json:"sha256"`
	Invalidated bool   `json:"invalidated"`
	Fact        string `json:"fact"`
}

// EntityMergePreview is both the dry-run report and the apply receipt.
type EntityMergePreview struct {
	ID                  string                       `json:"id,omitempty"`
	Survivor            string                       `json:"survivor"`
	Retire              []string                     `json:"retire"`
	Entities            map[string]Entity            `json:"entities"`
	ProposedMetadata    Entity                       `json:"proposed_metadata"`
	Expected            EntityMergeExpected          `json:"expected"`
	FactFingerprints    []EntityMergeFactFingerprint `json:"fact_fingerprints"`
	FactsTouched        int                          `json:"facts_touched"`
	CurrentFacts        int                          `json:"current_facts"`
	InvalidatedFacts    int                          `json:"invalidated_facts"`
	RequiredAliases     []string                     `json:"required_aliases"`
	DroppedAliases      []EntityMergeAliasDrop       `json:"dropped_aliases,omitempty"`
	RepoRefs            []string                     `json:"repo_refs"`
	SelfLoops           []string                     `json:"self_loops,omitempty"`
	FactKeyCollisions   []string                     `json:"fact_key_collisions,omitempty"`
	ExternalAliasOwners []string                     `json:"external_alias_owners,omitempty"`
	ExternalListings    []string                     `json:"external_listings,omitempty"`
	Problems            []string                     `json:"problems,omitempty"`
	Ready               bool                         `json:"ready"`
	Applied             bool                         `json:"applied"`
}

type entityMergeAnalysis struct {
	preview          EntityMergePreview
	metadata         Entity
	entities         map[string]Entity
	facts            []Fact
	updatedFacts     []Fact
	aliasClaims      map[string]string
	requiredNorms    map[string]string // normalized -> representative spelling
	rehomeNorms      map[string]string // normalized -> reviewed outside owner
	fingerprintNorms map[string]string
	group            map[string]bool
}

// PreviewEntityMerge snapshots and validates a group without writing.
func (s *Store) PreviewEntityMerge(req EntityMergeRequest) (EntityMergePreview, error) {
	var analysis entityMergeAnalysis
	err := s.db.View(func(txn *badger.Txn) error {
		var err error
		analysis, err = analyzeEntityMergeTxn(txn, req)
		return err
	})
	return analysis.preview, err
}

// MergeEntities applies one group in a single Badger transaction. Every
// entity, fact, and relevant alias claim is fingerprinted again inside the
// write transaction. Unrepresentable merges (self-loops or duplicate fact
// keys) are refused; timestamps are never nudged and facts are never dropped.
func (s *Store) MergeEntities(req EntityMergeRequest) (EntityMergePreview, error) {
	var analysis entityMergeAnalysis
	err := s.db.Update(func(txn *badger.Txn) error {
		var err error
		analysis, err = analyzeEntityMergeTxn(txn, req)
		if err != nil {
			return err
		}
		if err := verifyMergeExpected(req.Expected, analysis.preview.Expected); err != nil {
			return err
		}
		if req.Metadata == nil {
			return errors.New("memory: merge metadata is required for apply")
		}
		if !analysis.preview.Ready {
			return fmt.Errorf("memory: merge %s is not ready: %s", mergeLabel(req), strings.Join(analysis.preview.Problems, "; "))
		}

		// Remove every old fact key and reverse edge first. All replacement
		// keys were proven unique by the analysis above.
		for _, f := range analysis.facts {
			if err := txn.Delete(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom)); err != nil {
				return err
			}
			if f.Dst != "" {
				if err := txn.Delete(adjKey(f.Dst, f.Src, f.Relation, f.ValidFrom)); err != nil {
					return err
				}
			}
		}
		for _, f := range analysis.updatedFacts {
			b, err := json.Marshal(f)
			if err != nil {
				return err
			}
			if err := txn.Set(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom), b); err != nil {
				return err
			}
			if f.Dst != "" {
				if err := txn.Set(adjKey(f.Dst, f.Src, f.Relation, f.ValidFrom), nil); err != nil {
					return err
				}
			}
		}

		metadataJSON, err := json.Marshal(analysis.metadata)
		if err != nil {
			return err
		}
		if err := txn.Set([]byte(prefixEntity+req.Survivor), metadataJSON); err != nil {
			return err
		}
		for slug := range analysis.group {
			if slug != req.Survivor {
				if err := txn.Delete([]byte(prefixEntity + slug)); err != nil {
					return err
				}
			}
		}

		// Clear every routing key currently owned by a group member, then set
		// the complete reviewed spelling set to the survivor.
		for norm, owner := range analysis.aliasClaims {
			if analysis.group[owner] {
				if err := txn.Delete([]byte(prefixAlias + norm)); err != nil {
					return err
				}
			}
		}
		for norm := range analysis.requiredNorms {
			if err := txn.Set([]byte(prefixAlias+norm), []byte(req.Survivor)); err != nil {
				return err
			}
		}
		for norm, target := range analysis.rehomeNorms {
			if err := txn.Set([]byte(prefixAlias+norm), []byte(target)); err != nil {
				return err
			}
		}

		// Read-your-writes postconditions. A failed assertion aborts the same
		// transaction, so no partially merged group can commit.
		for slug := range analysis.group {
			if slug == req.Survivor {
				continue
			}
			if _, err := txn.Get([]byte(prefixEntity + slug)); !errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("memory: merge postcondition: retired entity %s still exists", slug)
			}
		}
		for norm := range analysis.requiredNorms {
			owner, found, err := aliasOwnerTxn(txn, norm)
			if err != nil || !found || owner != req.Survivor {
				return fmt.Errorf("memory: merge postcondition: alias %q resolves to %q, found=%v: %w", norm, owner, found, err)
			}
		}
		for norm, target := range analysis.rehomeNorms {
			owner, found, err := aliasOwnerTxn(txn, norm)
			if err != nil || !found || owner != target {
				return fmt.Errorf("memory: merge postcondition: rehomed alias %q resolves to %q, found=%v: %w", norm, owner, found, err)
			}
		}
		for _, f := range analysis.updatedFacts {
			item, err := txn.Get(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom))
			if err != nil {
				return fmt.Errorf("memory: merge postcondition: fact missing: %w", err)
			}
			var stored Fact
			if err := item.Value(func(value []byte) error { return json.Unmarshal(value, &stored) }); err != nil {
				return err
			}
			if !reflect.DeepEqual(stored, f) {
				return errors.New("memory: merge postcondition: rewritten fact changed")
			}
		}
		return nil
	})
	if err != nil {
		return analysis.preview, err
	}

	analysis.preview.Applied = true
	for i, old := range analysis.facts {
		s.notify(Event{Kind: "fact", Op: "delete", Fact: old})
		s.notify(Event{Kind: "fact", Op: "put", Fact: analysis.updatedFacts[i]})
	}
	s.notify(Event{Kind: "entity", Op: "put", Entity: analysis.metadata})
	for slug := range analysis.group {
		if slug != req.Survivor {
			s.notify(Event{Kind: "entity", Op: "delete", Slug: slug})
		}
	}
	return analysis.preview, nil
}

func analyzeEntityMergeTxn(txn *badger.Txn, req EntityMergeRequest) (entityMergeAnalysis, error) {
	a := entityMergeAnalysis{
		entities:         map[string]Entity{},
		aliasClaims:      map[string]string{},
		requiredNorms:    map[string]string{},
		rehomeNorms:      map[string]string{},
		fingerprintNorms: map[string]string{},
		group:            map[string]bool{},
	}
	a.preview = EntityMergePreview{ID: req.ID, Survivor: req.Survivor, Retire: append([]string(nil), req.Retire...), Entities: map[string]Entity{}}
	problem := func(msg string) { a.preview.Problems = append(a.preview.Problems, msg) }
	if strings.TrimSpace(req.Survivor) == "" {
		return a, errors.New("memory: merge survivor is required")
	}
	a.group[req.Survivor] = true
	for _, slug := range req.Retire {
		if strings.TrimSpace(slug) == "" || slug == req.Survivor || a.group[slug] {
			return a, fmt.Errorf("memory: merge has invalid retired slug %q", slug)
		}
		a.group[slug] = true
	}
	if len(req.Retire) == 0 {
		return a, errors.New("memory: merge requires at least one retired entity")
	}

	allEntities, err := entitiesTxn(txn)
	if err != nil {
		return a, err
	}
	for _, e := range allEntities {
		if a.group[e.Slug] {
			a.entities[e.Slug] = e
			a.preview.Entities[e.Slug] = e
		}
	}
	allEntitySlugs := make(map[string]bool, len(allEntities))
	allEntitiesBySlug := make(map[string]Entity, len(allEntities))
	for _, e := range allEntities {
		allEntitySlugs[e.Slug] = true
		allEntitiesBySlug[e.Slug] = e
	}
	for slug := range a.group {
		if _, ok := a.entities[slug]; !ok {
			problem("entity does not exist: " + slug)
		}
	}
	if len(a.entities) != len(a.group) {
		a.preview.Ready = false
		return a, nil
	}

	proposed := proposedMergeMetadata(req.Survivor, a.entities, req.Retire)
	if req.Metadata != nil {
		proposed = *req.Metadata
	}
	a.metadata = proposed
	a.preview.ProposedMetadata = proposed
	if proposed.Slug != req.Survivor {
		problem("metadata.slug must equal survivor")
	}
	if strings.TrimSpace(proposed.Name) == "" || strings.TrimSpace(proposed.Type) == "" {
		problem("metadata name and type are required")
	}

	claims, err := aliasClaimsTxn(txn)
	if err != nil {
		return a, err
	}
	a.aliasClaims = claims
	for norm, owner := range claims {
		if a.group[owner] {
			a.requiredNorms[norm] = norm
			a.fingerprintNorms[norm] = norm
		}
	}
	groupOrder := append([]string{req.Survivor}, req.Retire...)
	for _, slug := range groupOrder {
		e := a.entities[slug]
		for _, spelling := range append([]string{e.Slug, e.Name}, e.Aliases...) {
			if norm := Normalize(spelling); norm != "" {
				a.requiredNorms[norm] = spelling
				a.fingerprintNorms[norm] = spelling
			}
		}
	}
	dropped := map[string]bool{}
	for _, drop := range req.DropAliases {
		norm := Normalize(drop.Alias)
		if norm == "" || strings.TrimSpace(drop.Why) == "" {
			problem("every dropped alias requires a spelling and reason")
			continue
		}
		if dropped[norm] {
			problem("dropped alias is repeated: " + drop.Alias)
			continue
		}
		dropped[norm] = true
		a.fingerprintNorms[norm] = drop.Alias
		foundAlias := false
		protected := false
		for _, e := range a.entities {
			if Normalize(e.Name) == norm || Normalize(e.Slug) == norm {
				protected = true
			}
			for _, alias := range e.Aliases {
				if Normalize(alias) == norm {
					foundAlias = true
				}
			}
		}
		if protected {
			problem("entity names and slugs cannot be dropped: " + drop.Alias)
			continue
		}
		if !foundAlias {
			if owner, ok := claims[norm]; !ok || !a.group[owner] {
				problem("dropped spelling is not a group alias or claim: " + drop.Alias)
				continue
			}
		}
		var outsideListings []string
		for _, e := range allEntities {
			if !a.group[e.Slug] && entityListsNormalized(e, norm) {
				outsideListings = append(outsideListings, e.Slug)
			}
		}
		if len(outsideListings) > 0 && strings.TrimSpace(drop.RehomeTo) == "" {
			problem(fmt.Sprintf("dropped alias %q is listed outside the group and requires rehome_to", drop.Alias))
		}
		if strings.TrimSpace(drop.RehomeTo) != "" {
			target, ok := allEntitiesBySlug[drop.RehomeTo]
			if !ok || a.group[drop.RehomeTo] {
				problem(fmt.Sprintf("dropped alias %q has invalid rehome target %q", drop.Alias, drop.RehomeTo))
			} else if !entityListsNormalized(target, norm) {
				problem(fmt.Sprintf("rehome target %s does not list dropped alias %q", drop.RehomeTo, drop.Alias))
			} else {
				a.rehomeNorms[norm] = drop.RehomeTo
			}
		}
		delete(a.requiredNorms, norm)
		a.preview.DroppedAliases = append(a.preview.DroppedAliases, drop)
	}
	if req.Metadata == nil && len(dropped) > 0 {
		kept := proposed.Aliases[:0]
		for _, alias := range proposed.Aliases {
			if !dropped[Normalize(alias)] {
				kept = append(kept, alias)
			}
		}
		proposed.Aliases = kept
		a.metadata = proposed
		a.preview.ProposedMetadata = proposed
	}
	if req.Metadata == nil {
		// A stale routing key owned by either endpoint is part of the identity
		// snapshot even when no entity currently lists it. Surface it in the
		// proposed metadata so the reviewer must consciously keep it (or first
		// remove it with the reviewed unalias/rehome operation).
		proposedNorms := normalizedNameSet(proposed.Name, proposed.Aliases)
		proposedNorms[Normalize(proposed.Slug)] = true
		var stale []string
		for norm, spelling := range a.requiredNorms {
			if !proposedNorms[norm] {
				stale = append(stale, spelling)
				proposedNorms[norm] = true
			}
		}
		sort.Strings(stale)
		proposed.Aliases = append(proposed.Aliases, stale...)
		a.metadata = proposed
		a.preview.ProposedMetadata = proposed
	}
	for _, spelling := range append([]string{proposed.Slug, proposed.Name}, proposed.Aliases...) {
		if norm := Normalize(spelling); norm != "" {
			a.requiredNorms[norm] = spelling
			a.fingerprintNorms[norm] = spelling
		}
	}

	metadataNorms := normalizedNameSet(proposed.Name, proposed.Aliases)
	metadataNorms[Normalize(proposed.Slug)] = true
	seenMetadataNorms := map[string]bool{Normalize(proposed.Name): true, Normalize(proposed.Slug): true}
	for _, alias := range proposed.Aliases {
		norm := Normalize(alias)
		if norm == "" || seenMetadataNorms[norm] {
			problem("metadata aliases are not uniquely normalized: " + alias)
			continue
		}
		seenMetadataNorms[norm] = true
	}
	for norm, spelling := range a.requiredNorms {
		if !metadataNorms[norm] {
			problem(fmt.Sprintf("metadata drops spelling %q (%s)", spelling, norm))
		}
		if owner, ok := claims[norm]; ok && !a.group[owner] {
			a.preview.ExternalAliasOwners = append(a.preview.ExternalAliasOwners, norm+" -> "+owner)
		}
	}
	for norm := range dropped {
		if metadataNorms[norm] {
			problem("metadata keeps explicitly dropped alias " + norm)
		}
	}
	requiredRefs := map[string]bool{}
	for _, slug := range groupOrder {
		for _, ref := range a.entities[slug].RepoRefs {
			requiredRefs[ref] = true
		}
	}
	seenMetadataRefs := map[string]bool{}
	for _, ref := range proposed.RepoRefs {
		if seenMetadataRefs[ref] {
			problem("metadata repo refs are not unique: " + ref)
		}
		seenMetadataRefs[ref] = true
	}
	for ref := range requiredRefs {
		if !seenMetadataRefs[ref] {
			problem("metadata drops repo ref " + ref)
		}
	}
	for _, e := range allEntities {
		if a.group[e.Slug] {
			continue
		}
		listed := normalizedNameSet(e.Name, e.Aliases)
		listed[Normalize(e.Slug)] = true
		for norm := range a.requiredNorms {
			if listed[norm] {
				a.preview.ExternalListings = append(a.preview.ExternalListings, norm+" listed by "+e.Slug)
			}
		}
	}
	if len(a.preview.ExternalAliasOwners) > 0 {
		problem("a required spelling is owned outside the merge group")
	}
	if len(a.preview.ExternalListings) > 0 {
		problem("a required spelling is listed outside the merge group")
	}

	allFacts, err := factsTxn(txn)
	if err != nil {
		return a, err
	}
	for _, f := range allFacts {
		if !a.group[f.Src] && !a.group[f.Dst] {
			continue
		}
		a.facts = append(a.facts, f)
		updated := f
		if a.group[updated.Src] {
			updated.Src = req.Survivor
		}
		if a.group[updated.Dst] {
			updated.Dst = req.Survivor
		}
		a.updatedFacts = append(a.updatedFacts, updated)
		if f.InvalidAt == nil {
			a.preview.CurrentFacts++
		} else {
			a.preview.InvalidatedFacts++
		}
		key := string(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom))
		a.preview.FactFingerprints = append(a.preview.FactFingerprints, EntityMergeFactFingerprint{
			Key: key, SHA256: hashJSON(f), Invalidated: f.InvalidAt != nil, Fact: f.Fact,
		})
	}
	a.preview.FactsTouched = len(a.facts)
	if len(a.facts) == 0 {
		problem("merge would leave a hollow survivor with no facts")
	}
	targetKeys := map[string]string{}
	for i, f := range a.updatedFacts {
		oldKey := a.preview.FactFingerprints[i].Key
		newKey := string(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom))
		if f.Dst != "" && f.Src == f.Dst {
			a.preview.SelfLoops = append(a.preview.SelfLoops, oldKey+" -> "+newKey)
		}
		if !allEntitySlugs[f.Src] && f.Src != req.Survivor {
			problem("rewritten fact has dangling source " + f.Src)
		}
		if f.Dst != "" && !allEntitySlugs[f.Dst] && f.Dst != req.Survivor {
			problem("rewritten fact has dangling destination " + f.Dst)
		}
		if prior, ok := targetKeys[newKey]; ok && prior != oldKey {
			a.preview.FactKeyCollisions = append(a.preview.FactKeyCollisions, prior+" + "+oldKey+" -> "+newKey)
		} else {
			targetKeys[newKey] = oldKey
		}
	}
	if len(a.preview.SelfLoops) > 0 {
		problem("merge would create self-loop facts")
	}
	if len(a.preview.FactKeyCollisions) > 0 {
		problem("merge would create duplicate fact keys")
	}

	entityHashes := map[string]string{}
	for slug, e := range a.entities {
		entityHashes[slug] = hashJSON(e)
	}
	a.preview.Expected = EntityMergeExpected{
		Entities:    entityHashes,
		Facts:       hashJSON(a.facts),
		AliasClaims: hashAliasSubset(a.fingerprintNorms, claims),
	}
	for _, spelling := range a.requiredNorms {
		a.preview.RequiredAliases = append(a.preview.RequiredAliases, spelling)
	}
	sort.Strings(a.preview.RequiredAliases)
	a.preview.RepoRefs = append([]string(nil), proposed.RepoRefs...)
	sort.Strings(a.preview.ExternalAliasOwners)
	sort.Strings(a.preview.ExternalListings)
	sort.Strings(a.preview.Problems)
	a.preview.Ready = len(a.preview.Problems) == 0
	return a, nil
}

func proposedMergeMetadata(survivor string, entities map[string]Entity, retire []string) Entity {
	out := entities[survivor]
	originalAliases := append([]string(nil), out.Aliases...)
	out.Aliases = nil
	seenAliases := map[string]bool{Normalize(out.Name): true, Normalize(out.Slug): true}
	for _, alias := range originalAliases {
		norm := Normalize(alias)
		if norm != "" && !seenAliases[norm] {
			seenAliases[norm] = true
			out.Aliases = append(out.Aliases, alias)
		}
	}
	seenRefs := map[string]bool{}
	originalRefs := append([]string(nil), out.RepoRefs...)
	out.RepoRefs = nil
	for _, ref := range originalRefs {
		if !seenRefs[ref] {
			seenRefs[ref] = true
			out.RepoRefs = append(out.RepoRefs, ref)
		}
	}
	order := append([]string{survivor}, retire...)
	for _, slug := range order {
		e := entities[slug]
		if out.Description == "" && e.Description != "" {
			out.Description = e.Description
		}
		if out.CreatedAt.IsZero() || (!e.CreatedAt.IsZero() && e.CreatedAt.Before(out.CreatedAt)) {
			out.CreatedAt = e.CreatedAt
		}
		if e.LastSeen.After(out.LastSeen) {
			out.LastSeen = e.LastSeen
		}
		for _, spelling := range append([]string{e.Name, e.Slug}, e.Aliases...) {
			norm := Normalize(spelling)
			if norm == "" || seenAliases[norm] {
				continue
			}
			seenAliases[norm] = true
			out.Aliases = append(out.Aliases, spelling)
		}
		for _, ref := range e.RepoRefs {
			if !seenRefs[ref] {
				seenRefs[ref] = true
				out.RepoRefs = append(out.RepoRefs, ref)
			}
		}
	}
	sort.Strings(out.RepoRefs)
	return out
}

func entityListsNormalized(e Entity, norm string) bool {
	if Normalize(e.Slug) == norm || Normalize(e.Name) == norm {
		return true
	}
	for _, alias := range e.Aliases {
		if Normalize(alias) == norm {
			return true
		}
	}
	return false
}

func verifyMergeExpected(got, want EntityMergeExpected) error {
	if got.Facts == "" || got.AliasClaims == "" || len(got.Entities) == 0 {
		return errors.New("memory: merge expected entity, fact, and alias fingerprints are required for apply")
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("memory: merge snapshot changed: got %+v, want %+v", want, got)
	}
	return nil
}

func entitiesTxn(txn *badger.Txn) ([]Entity, error) {
	var out []Entity
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	pb := []byte(prefixEntity)
	for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
		var e Entity
		if err := it.Item().Value(func(value []byte) error { return json.Unmarshal(value, &e) }); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func factsTxn(txn *badger.Txn) ([]Fact, error) {
	var out []Fact
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	pb := []byte(prefixFact)
	for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
		var f Fact
		if err := it.Item().Value(func(value []byte) error { return json.Unmarshal(value, &f) }); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func aliasClaimsTxn(txn *badger.Txn) (map[string]string, error) {
	out := map[string]string{}
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	pb := []byte(prefixAlias)
	for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
		key := string(it.Item().KeyCopy(nil))
		value, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		out[strings.TrimPrefix(key, prefixAlias)] = string(value)
	}
	return out, nil
}

func hashAliasSubset(required map[string]string, claims map[string]string) string {
	type claim struct{ Norm, Owner string }
	rows := make([]claim, 0, len(required))
	for norm := range required {
		rows = append(rows, claim{Norm: norm, Owner: claims[norm]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Norm < rows[j].Norm })
	return hashJSON(rows)
}

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func mergeLabel(req EntityMergeRequest) string {
	if req.ID != "" {
		return req.ID
	}
	return req.Survivor
}
