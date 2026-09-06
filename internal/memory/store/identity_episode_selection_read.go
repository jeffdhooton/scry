package store

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"
)

type episodeSelectionToken struct {
	Selected   bool   `json:"selected"`
	HeadKey    string `json:"head_key"`
	HeadDigest string `json:"head_digest"`
	Revision   uint64 `json:"revision"`
	ResultKey  string `json:"result_key"`
}

type episodeResultChunk struct {
	Key        string `json:"key"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"next_offset"`
	Data       []byte `json:"data"`
}

func readEpisodeSelection(st *Store, id string) (episodeSelection, error) {
	if id == "" || !utf8.ValidString(id) {
		return episodeSelection{}, errEpisodeSelection
	}
	var result episodeSelection
	err := inputView(st, func(tx *Store) error {
		var err error
		result, _, err = readEpisodeSelectionTxn(tx, id)
		return err
	})
	if err != nil {
		return episodeSelection{}, errEpisodeSelection
	}
	return result, nil
}

// Inspection identity only; digest is NEVER the writer's exact raw-head CAS.
func readEpisodeSelectionToken(st *Store, id string, maxBytes int) (episodeSelectionToken, error) {
	if maxBytes < 512 || maxBytes > 24576 {
		return episodeSelectionToken{}, errEpisodeSelection
	}
	r, err := readEpisodeSelection(st, id)
	if err != nil {
		return episodeSelectionToken{}, errEpisodeSelection
	}
	token := episodeSelectionToken{}
	if r.Selected {
		token = episodeSelectionToken{Selected: true, HeadKey: episodeHeadKey(id), HeadDigest: generationDigest(r.HeadRaw), Revision: r.Head.Revision, ResultKey: r.Head.ResultKey}
	}
	raw, err := json.Marshal(token)
	if err != nil || len(raw) > maxBytes {
		return episodeSelectionToken{}, errEpisodeSelection
	}
	return token, nil
}

// Every chunk pins an immutable result, not whatever a later head selects.
func readEpisodeResultChunk(st *Store, id, key string, offset, maxBytes int) (episodeResultChunk, error) {
	if id == "" || !utf8.ValidString(id) || offset < 0 || maxBytes < 512 || maxBytes > 24576 {
		return episodeResultChunk{}, errEpisodeSelection
	}
	var chunk episodeResultChunk
	err := inputView(st, func(tx *Store) error {
		_, raw, _, _, err := readEpisodeResultRow(tx, key, id)
		if err != nil || offset > len(raw) {
			return errEpisodeSelection
		}
		chunk = episodeResultChunk{Key: key, Offset: offset, NextOffset: -1, Data: []byte{}}
		lo, hi := 0, len(raw)-offset
		if hi > maxBytes {
			hi = maxBytes
		}
		for lo < hi {
			mid := (lo + hi + 1) / 2
			trial := chunk
			trial.Data = raw[offset : offset+mid]
			if offset+mid < len(raw) {
				trial.NextOffset = offset + mid
			}
			encoded, _ := json.Marshal(trial)
			if len(encoded) <= maxBytes {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		if lo == 0 && offset < len(raw) {
			return errEpisodeSelection
		}
		chunk.Data = bytes.Clone(raw[offset : offset+lo])
		if offset+lo < len(raw) {
			chunk.NextOffset = offset + lo
		}
		encoded, _ := json.Marshal(chunk)
		if len(encoded) > maxBytes {
			return errEpisodeSelection
		}
		return nil
	})
	if err != nil {
		return episodeResultChunk{}, errEpisodeSelection
	}
	return chunk, nil
}
