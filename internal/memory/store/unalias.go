package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

// AliasDrop is an explicit spelling disposition, never an inferred owner.
type AliasDrop struct {
	Entity   string `json:"entity"`
	Alias    string `json:"alias"`
	RehomeTo string `json:"rehome_to,omitempty"`
	Why      string `json:"why,omitempty"`
}

// AliasRepairExpected pins the review's identity, fact, listing and routing
// evidence. In particular, an alias claim can change without changing either
// entity. Listings include outside entities even when they have no facts.
type AliasRepairExpected struct {
	Plan         string              `json:"plan"`
	Entities     map[string]string   `json:"entities"`
	Facts        string              `json:"facts"`
	AliasClaims  map[string]string   `json:"alias_claims"`
	ClaimPresent map[string]bool     `json:"claim_present"`
	Listings     map[string][]string `json:"listings"`
	Rejections   string              `json:"rejections"`
}

type AliasRepairRequest struct {
	Drops    []AliasDrop          `json:"drops"`
	Expected *AliasRepairExpected `json:"expected,omitempty"`
}

type AliasRepairPreview struct {
	Expected           AliasRepairExpected         `json:"expected"`
	Entities           map[string]Entity           `json:"entities"`
	ProposedEntities   map[string]Entity           `json:"proposed_entities"`
	ProposedRejections map[string][]AliasRejection `json:"proposed_rejections"`
	FactsTouched       int                         `json:"facts_touched"`
	Problems           []string                    `json:"problems,omitempty"`
	Ready              bool                        `json:"ready"`
	Applied            bool                        `json:"applied"`
}

type aliasRepairAnalysis struct {
	preview           AliasRepairPreview
	entities          []Entity
	facts             []Fact
	claims            map[string]string
	updated           map[string]Entity
	updatedClaims     map[string]string
	rejections        map[string][]AliasRejection
	updatedRejections map[string][]AliasRejection
}

func entityListsAlias(e Entity, norm string) bool {
	return Normalize(e.Slug) == norm || normalizedNameSet(e.Name, e.Aliases)[norm]
}

func analyzeAliasRepair(txn *badger.Txn, req AliasRepairRequest) (aliasRepairAnalysis, error) {
	a := aliasRepairAnalysis{}
	var err error
	if a.entities, err = entitiesTxn(txn); err != nil {
		return a, err
	}
	if a.facts, err = factsTxn(txn); err != nil {
		return a, err
	}
	if a.claims, err = aliasClaimsTxn(txn); err != nil {
		return a, err
	}
	if a.rejections, err = aliasRejectionsTxn(txn); err != nil {
		return a, err
	}
	a.updatedRejections = cloneAliasRejections(a.rejections)
	p := &a.preview
	p.Expected = AliasRepairExpected{Plan: hashJSON(req.Drops), Entities: map[string]string{}, AliasClaims: map[string]string{}, ClaimPresent: map[string]bool{}, Listings: map[string][]string{}}
	p.Entities = map[string]Entity{}
	p.ProposedEntities = map[string]Entity{}
	bySlug := map[string]Entity{}
	for _, e := range a.entities {
		bySlug[e.Slug] = e
	}
	participants := map[string]bool{}
	removals := map[string]map[string]bool{}
	rehomes := map[string]string{}
	norms := map[string]bool{}
	seen := map[string]bool{}
	problem := func(s string) { p.Problems = append(p.Problems, s) }
	if len(req.Drops) == 0 {
		problem("no drops given")
	}
	for _, drop := range req.Drops {
		norm := Normalize(drop.Alias)
		if norm == "" || strings.TrimSpace(drop.Why) == "" {
			problem("every drop requires a nonempty alias and why")
		}
		norms[norm] = true
		participants[drop.Entity] = true
		e, exists := bySlug[drop.Entity]
		if !exists {
			problem("no entity " + drop.Entity)
			continue
		}
		if Normalize(e.Name) == norm || Normalize(e.Slug) == norm {
			problem("cannot drop an entity's own name: " + drop.Alias)
		}
		if !normalizedNameSet("", e.Aliases)[norm] {
			problem(drop.Entity + " does not list " + drop.Alias)
		}
		key := drop.Entity + "\x00" + drop.Alias
		if seen[key] {
			problem("duplicate drop " + drop.Entity + ": " + drop.Alias)
		}
		seen[key] = true
		if removals[drop.Entity] == nil {
			removals[drop.Entity] = map[string]bool{}
		}
		removals[drop.Entity][norm] = true
		addAliasRejection(a.updatedRejections, AliasRejection{Entity: drop.Entity, Alias: drop.Alias, Why: drop.Why, Plan: p.Expected.Plan})
		if drop.RehomeTo != "" {
			participants[drop.RehomeTo] = true
			target, found := bySlug[drop.RehomeTo]
			if !found || drop.RehomeTo == drop.Entity || !entityListsAlias(target, norm) {
				problem("rehome target does not already list " + drop.Alias + ": " + drop.RehomeTo)
			}
			if prior := rehomes[norm]; prior != "" && prior != drop.RehomeTo {
				problem("conflicting rehomes for " + drop.Alias)
			}
			rehomes[norm] = drop.RehomeTo
		}
	}
	a.updated = map[string]Entity{}
	for slug, removed := range removals {
		e := bySlug[slug]
		var kept []string // empty aliases round-trip through omitempty as nil
		for _, alias := range e.Aliases {
			if !removed[Normalize(alias)] {
				kept = append(kept, alias)
			}
		}
		e.Aliases = kept
		a.updated[slug] = e
		p.ProposedEntities[slug] = e
	}
	a.updatedClaims = map[string]string{}
	for norm, owner := range a.claims {
		a.updatedClaims[norm] = owner
	}
	for norm := range norms {
		owner, present := a.claims[norm]
		p.Expected.ClaimPresent[norm] = present
		p.Expected.AliasClaims[norm] = owner // empty explicitly fingerprints absence
		p.Expected.Listings[norm] = []string{}
		if _, exists := bySlug[owner]; exists {
			participants[owner] = true
		}
		remaining := []string{}
		for _, e := range a.entities {
			if !entityListsAlias(e, norm) {
				continue
			}
			participants[e.Slug] = true
			p.Expected.Listings[norm] = append(p.Expected.Listings[norm], e.Slug)
			after := e
			if changed, ok := a.updated[e.Slug]; ok {
				after = changed
			}
			if entityListsAlias(after, norm) {
				remaining = append(remaining, e.Slug)
			}
		}
		sort.Strings(p.Expected.Listings[norm])
		if target := rehomes[norm]; target != "" {
			if len(a.updatedRejections[aliasRejectionKey(target, norm)]) > 0 {
				problem("rehome target has a reviewed rejection for " + norm + ": " + target)
			}
			e := bySlug[target]
			if changed, ok := a.updated[target]; ok {
				e = changed
			}
			if !entityListsAlias(e, norm) {
				problem("rehome target loses its listing in this batch: " + norm)
			}
			a.updatedClaims[norm] = target
		} else if removals[owner][norm] {
			if len(remaining) > 0 {
				problem("dropping " + norm + " would strand other listings; explicitly rehome or review every listing")
			}
			delete(a.updatedClaims, norm)
		} else if owner == "" && len(remaining) > 0 {
			problem("unindexed remaining listing needs an explicit rehome: " + norm)
		}
	}
	for slug := range participants {
		if e, exists := bySlug[slug]; exists {
			p.Entities[slug] = e
			p.Expected.Entities[slug] = hashJSON(e)
		}
	}
	touching := []Fact{}
	for _, f := range a.facts {
		if participants[f.Src] || (f.Dst != "" && participants[f.Dst]) {
			touching = append(touching, f)
		}
	}
	p.FactsTouched = len(touching)
	p.Expected.Facts = hashJSON(touching)
	p.Expected.Rejections = hashJSON(rejectionSubset(a.rejections, participants))
	p.ProposedRejections = rejectionSubset(a.updatedRejections, participants)
	for slug, e := range a.updated {
		for _, spelling := range append([]string{e.Slug, e.Name}, e.Aliases...) {
			if len(a.updatedRejections[aliasRejectionKey(slug, Normalize(spelling))]) > 0 {
				problem("updated entity still lists reviewed rejected spelling: " + slug + ": " + spelling)
			}
		}
	}
	if req.Expected != nil && !reflect.DeepEqual(*req.Expected, p.Expected) {
		problem("reviewed alias repair fingerprints changed")
	}
	p.Ready = len(p.Problems) == 0
	return a, nil
}

func (s *Store) PreviewAliasRepair(req AliasRepairRequest) (AliasRepairPreview, error) {
	var a aliasRepairAnalysis
	err := s.view(func(txn *badger.Txn) error { var err error; a, err = analyzeAliasRepair(txn, req); return err })
	return a.preview, err
}

// BackupAndRepairAliases serializes normal writers from backup through apply.
// Every row is checked against one original snapshot before any row writes;
// the complete batch then commits atomically. No entity is created or retired.
func (s *Store) BackupAndRepairAliases(w durableBackupWriter, req AliasRepairRequest) (uint64, AliasRepairPreview, error) {
	if err := s.refuseAdmissionMaintenance(); err != nil {
		return 0, AliasRepairPreview{}, err
	}
	var n uint64
	var preview AliasRepairPreview
	err := func() error {
		s.maintenanceMu.Lock()
		defer s.maintenanceMu.Unlock()
		var err error
		n, preview, err = s.backupAndRepairAliasesUnlocked(w, req)
		return err
	}()
	if err == nil {
		slugs := make([]string, 0, len(preview.ProposedEntities))
		for slug := range preview.ProposedEntities {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		for _, slug := range slugs {
			s.notify(Event{Kind: "entity", Op: "put", Entity: preview.ProposedEntities[slug]})
		}
	}
	return n, preview, err
}

func (s *Store) backupAndRepairAliasesUnlocked(w durableBackupWriter, req AliasRepairRequest) (uint64, AliasRepairPreview, error) {
	n, err := s.backupUnlocked(w)
	if err != nil {
		_ = w.Close()
		return n, AliasRepairPreview{}, err
	}
	if err := w.Sync(); err != nil {
		_ = w.Close()
		return n, AliasRepairPreview{}, fmt.Errorf("sync alias backup: %w", err)
	}
	if err := w.Close(); err != nil {
		return n, AliasRepairPreview{}, fmt.Errorf("close alias backup: %w", err)
	}
	var a aliasRepairAnalysis
	err = s.db.Update(func(txn *badger.Txn) error {
		var err error
		a, err = analyzeAliasRepair(txn, req)
		if err != nil {
			return err
		}
		if req.Expected == nil {
			return errors.New("alias repair apply requires reviewed fingerprints")
		}
		if !a.preview.Ready {
			return fmt.Errorf("alias repair refused: %s", strings.Join(a.preview.Problems, "; "))
		}
		for slug, e := range a.updated {
			b, err := json.Marshal(e)
			if err != nil {
				return err
			}
			if err := txn.Set([]byte(prefixEntity+slug), b); err != nil {
				return err
			}
		}
		for norm := range a.preview.Expected.AliasClaims {
			key := []byte(prefixAlias + norm)
			if owner, exists := a.updatedClaims[norm]; exists {
				err = txn.Set(key, []byte(owner))
			} else {
				err = txn.Delete(key)
			}
			if err != nil {
				return err
			}
		}
		if err := writeAliasRejectionsTxn(txn, a.rejections, a.updatedRejections); err != nil {
			return err
		}
		// Exact postconditions cover the entire entity/fact/alias state, not
		// only the edited keys. Metadata outside reviewed aliases is immutable.
		expectedEntities := append([]Entity(nil), a.entities...)
		for i, e := range expectedEntities {
			if changed, ok := a.updated[e.Slug]; ok {
				expectedEntities[i] = changed
			}
		}
		entities, err := entitiesTxn(txn)
		if err != nil {
			return err
		}
		facts, err := factsTxn(txn)
		if err != nil {
			return err
		}
		claims, err := aliasClaimsTxn(txn)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(entities, expectedEntities) || !reflect.DeepEqual(facts, a.facts) || !reflect.DeepEqual(claims, a.updatedClaims) {
			return errors.New("alias repair postcondition failed")
		}
		return nil
	})
	if err == nil {
		a.preview.Applied = true
	}
	return n, a.preview, err
}
