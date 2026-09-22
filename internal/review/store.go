package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type diskState struct {
	Version  int                `json:"version"`
	Records  []Record           `json:"records"`
	Daily    map[string]int     `json:"daily"`
	Reserved map[string]float64 `json:"reserved"`
	Seen     map[string]bool    `json:"seen"`
	Blocked  string             `json:"blocked,omitempty"`
}

func loadState(home string) (diskState, error) {
	s := diskState{Version: 1, Records: []Record{}, Daily: map[string]int{}, Reserved: map[string]float64{}, Seen: map[string]bool{}}
	p := filepath.Join(home, "state.json")
	info, e := os.Stat(p)
	if errors.Is(e, os.ErrNotExist) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	if info.Size() > 64<<20 {
		return s, errors.New("review state exceeds 64 MiB")
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return s, e
	}
	// A present file must carry every durable ledger field. Defaults are only
	// for a genuinely new store, never a partially written/corrupt ledger.
	s = diskState{}
	if e = json.Unmarshal(b, &s); e != nil {
		return s, fmt.Errorf("invalid review state: %w", e)
	}
	if s.Version != 1 || s.Records == nil || s.Daily == nil || s.Reserved == nil || s.Seen == nil {
		return s, errors.New("unsupported or incomplete review state; refusing to reset usage")
	}
	return s, nil
}
func (s *Service) saveLocked() error {
	if len(s.state.Records) > s.opts.Retain {
		s.state.Records = append([]Record(nil), s.state.Records[len(s.state.Records)-s.opts.Retain:]...)
	}
	if e := os.MkdirAll(s.home, 0700); e != nil {
		return e
	}
	b, e := json.Marshal(s.state)
	if e != nil {
		return e
	}
	if len(b) > 64<<20 {
		return errors.New("review state exceeds 64 MiB")
	}
	f, e := os.CreateTemp(s.home, ".state-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(tmp, filepath.Join(s.home, "state.json")); e != nil {
		return e
	}
	d, e := os.Open(s.home)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
