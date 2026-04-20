package index

import (
	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

const maxFilesPerCommitForCochange = 50

func computeCochange(st *gitstore.Store) error {
	commits, err := st.GetRecentCommits(0)
	if err != nil {
		return err
	}

	type pairKey struct{ a, b string }
	pairs := map[pairKey]*gitstore.CochangeRecord{}

	for _, c := range commits {
		if len(c.Files) < 2 || len(c.Files) > maxFilesPerCommitForCochange {
			continue
		}
		for i := 0; i < len(c.Files); i++ {
			for j := i + 1; j < len(c.Files); j++ {
				a, b := c.Files[i].Path, c.Files[j].Path
				if a > b {
					a, b = b, a
				}
				key := pairKey{a, b}
				if pairs[key] == nil {
					pairs[key] = &gitstore.CochangeRecord{FileA: a, FileB: b}
				}
				pairs[key].Count++
				if c.Date > pairs[key].LastChanged {
					pairs[key].LastChanged = c.Date
				}
			}
		}
	}

	w := st.NewWriter()
	for k, rec := range pairs {
		if err := w.PutCochange(k.a, k.b, rec); err != nil {
			return err
		}
	}
	return w.Flush()
}
