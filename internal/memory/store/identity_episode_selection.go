package store

// PRIVATE, UNCALLED structural episode selection. Descriptions are not support,
// registration completeness, ownership, successful ingestion or commit proof.

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

const identityEpisodePrefix = "io-episode:"
const identityEpisodeHeadPrefix = "io-head:"

var errEpisodeSelection = errors.New("memory: invalid episode selection")

type episodeDeclaration struct {
	Ordinal        int      `json:"ordinal"`
	ObservationKey string   `json:"observation_key"`
	Kind           string   `json:"kind"`
	Births         []string `json:"births"`
}

type episodeAssertion struct {
	Ordinal                int    `json:"ordinal"`
	Kind                   string `json:"kind"`
	Reason                 string `json:"reason"`
	SourceObservation      string `json:"source_observation"`
	DestinationObservation string `json:"destination_observation"`
	OutcomeKey             string `json:"outcome_key"`
}

type episodeBirth struct {
	Birth            identityBirth `json:"birth"`
	FirstObservation string        `json:"first_observation"`
	OutcomeKey       string        `json:"outcome_key"`
}

type episodeResult struct {
	Version      int                  `json:"version"`
	EpisodeID    string               `json:"episode_id"`
	InputKey     string               `json:"input_key"`
	Declarations []episodeDeclaration `json:"declarations"`
	Assertions   []episodeAssertion   `json:"assertions"`
	Births       []episodeBirth       `json:"births"`
	Predecessor  string               `json:"predecessor"`
}

type episodeHead struct {
	Version   int    `json:"version"`
	EpisodeID string `json:"episode_id"`
	ResultKey string `json:"result_key"`
	Revision  uint64 `json:"revision"`
}

type episodeCounts struct {
	Committed           int `json:"committed"`
	Deferred            int `json:"deferred"`
	NonAssertion        int `json:"non_assertion"`
	Unresolved          int `json:"unresolved"`
	DeclarationDeferred int `json:"declaration_deferred"`
	BirthSupported      int `json:"birth_supported"`
	BirthNoAssertion    int `json:"birth_no_assertion"`
}

type episodeSelection struct {
	Selected  bool          `json:"selected"`
	Head      episodeHead   `json:"head"`
	HeadRaw   []byte        `json:"head_raw"`
	Result    episodeResult `json:"result"`
	ResultRaw []byte        `json:"result_raw"`
	Counts    episodeCounts `json:"counts"`
}

type episodeHeadExpectation struct {
	Exists bool
	Raw    []byte
}
type episodeProposal struct {
	Result   episodeResult
	Outcomes []identityOutcome
}

func episodeResultPrefix(id string) string {
	return identityEpisodePrefix + generationDigest([]byte(id)) + ":"
}
func episodeHeadKey(id string) string {
	return identityEpisodeHeadPrefix + generationDigest([]byte(id))
}
func episodeResultKeyValid(id, key string) bool {
	prefix := episodeResultPrefix(id)
	return strings.HasPrefix(key, prefix) && legacyDigestValid(strings.TrimPrefix(key, prefix))
}

func episodeSelectionFailure(st *Store, err error) error {
	safe := error(errEpisodeSelection)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errEpisodeSelection, kind)
			break
		}
	}
	if st != nil && st.txn != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

func encodeEpisodeResult(r episodeResult) ([]byte, string, error) {
	if r.Version != 1 || r.EpisodeID == "" || !utf8.ValidString(r.EpisodeID) || r.InputKey == "" || r.Declarations == nil || r.Assertions == nil || r.Births == nil || (r.Predecessor != "" && !episodeResultKeyValid(r.EpisodeID, r.Predecessor)) {
		return nil, "", errEpisodeSelection
	}
	r.Births = append([]episodeBirth{}, r.Births...)
	for i, b := range r.Births {
		record, _, err := identityBirthRecord(b.Birth)
		if err != nil || record.Birth.EpisodeID != r.EpisodeID {
			return nil, "", errEpisodeSelection
		}
		r.Births[i].Birth = record.Birth
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, "", errEpisodeSelection
	}
	var decoded episodeResult
	if json.Unmarshal(raw, &decoded) != nil || !reflect.DeepEqual(r, decoded) {
		return nil, "", errEpisodeSelection
	}
	return raw, episodeResultPrefix(r.EpisodeID) + generationDigest(raw), nil
}

func decodeEpisodeResult(key string, raw []byte, id string) (episodeResult, error) {
	var r episodeResult
	if json.Unmarshal(raw, &r) != nil {
		return episodeResult{}, errEpisodeSelection
	}
	canonical, want, err := encodeEpisodeResult(r)
	if err != nil || r.EpisodeID != id || want != key || !bytes.Equal(raw, canonical) {
		return episodeResult{}, errEpisodeSelection
	}
	return r, nil
}

func encodeEpisodeHead(h episodeHead) ([]byte, error) {
	if h.Version != 1 || h.EpisodeID == "" || !utf8.ValidString(h.EpisodeID) || h.Revision == 0 || !episodeResultKeyValid(h.EpisodeID, h.ResultKey) {
		return nil, errEpisodeSelection
	}
	return json.Marshal(h)
}

func decodeEpisodeHead(key string, raw []byte, id string) (episodeHead, error) {
	var h episodeHead
	if json.Unmarshal(raw, &h) != nil {
		return episodeHead{}, errEpisodeSelection
	}
	canonical, err := encodeEpisodeHead(h)
	if err != nil || h.EpisodeID != id || key != episodeHeadKey(id) || !bytes.Equal(raw, canonical) {
		return episodeHead{}, errEpisodeSelection
	}
	return h, nil
}

type episodeBasis struct {
	st           *Store
	input        identityInputRevision
	key          string
	raw          []byte
	observations map[string]identityObservation
}

func newEpisodeBasis(st *Store, key, id string) (*episodeBasis, error) {
	r, raw, err := readInputRow(st, key, id)
	if err != nil {
		return nil, errEpisodeSelection
	}
	return &episodeBasis{st: st, input: r, key: key, raw: raw, observations: map[string]identityObservation{}}, nil
}

func (b *episodeBasis) observation(key string) (identityObservation, error) {
	if o, ok := b.observations[key]; ok {
		return o, nil
	}
	raw, exists, err := generationRead(b.st, []byte(key))
	if err != nil || !exists || matchRevisionObservation(b.key, b.raw, key, raw, b.input.EpisodeID) != nil {
		return identityObservation{}, errEpisodeSelection
	}
	o, err := decodeIdentityObservation(key, raw, b.input.EpisodeID)
	if err != nil || observationProvenance(b.st, o) != nil {
		return identityObservation{}, errEpisodeSelection
	}
	b.observations[key] = o
	return o, nil
}

func observationOrder(o identityObservation) [3]int {
	origin, side := 0, 0
	if o.Origin == "endpoint" {
		origin = 1
	}
	if o.Side == "dst" {
		side = 1
	}
	return [3]int{origin, o.Ordinal, side}
}

func orderLess(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func selectionRoleOrder(role string) int {
	switch role {
	case "declaration":
		return 0
	case "primary-src":
		return 1
	case "primary-dst":
		return 2
	case "supersedes-src":
		return 3
	case "supersedes-dst":
		return 4
	}
	return -1
}

// Normalize only owned proposals. Persisted selected values must already match.
func (b *episodeBasis) outcome(o identityOutcome, requireOrder bool) (identityOutcome, error) {
	raw, key, err := encodeIdentityOutcome(o)
	if err != nil {
		return identityOutcome{}, errEpisodeSelection
	}
	o, err = decodeIdentityOutcome(key, raw, b.input.EpisodeID)
	if err != nil || validateOutcomeLinks(b.st, o) != nil {
		return identityOutcome{}, errEpisodeSelection
	}
	original := append([]outcomeLink{}, o.Links...)
	orders := map[string][3]int{}
	for _, l := range o.Links {
		input, err := b.observation(l.Key)
		if err != nil {
			return identityOutcome{}, errEpisodeSelection
		}
		orders[l.Key] = observationOrder(input)
	}
	sort.Slice(o.Links, func(i, j int) bool {
		a, c := orders[o.Links[i].Key], orders[o.Links[j].Key]
		if a != c {
			return orderLess(a, c)
		}
		return selectionRoleOrder(o.Links[i].Role) < selectionRoleOrder(o.Links[j].Role)
	})
	if requireOrder && !reflect.DeepEqual(original, o.Links) {
		return identityOutcome{}, errEpisodeSelection
	}
	return o, nil
}

func sameEpisodeBirth(a, b identityBirth) bool {
	_, ar, ae := identityBirthRecord(a)
	_, br, be := identityBirthRecord(b)
	return ae == nil && be == nil && bytes.Equal(ar, br)
}

func outcomeHasLink(o identityOutcome, key, role string) bool {
	for _, l := range o.Links {
		if l.Key == key && l.Role == role {
			return true
		}
	}
	return false
}

func (b *episodeBasis) slot(key, origin string, ordinal int, side string) (identityObservation, error) {
	o, err := b.observation(key)
	if err != nil || o.Origin != origin || o.Ordinal != ordinal || o.Side != side {
		return identityObservation{}, errEpisodeSelection
	}
	return o, nil
}
