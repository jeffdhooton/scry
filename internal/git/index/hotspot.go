package index

import (
	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

func computeChurnAndContrib(st *gitstore.Store) error {
	commits, err := st.GetRecentCommits(0)
	if err != nil {
		return err
	}

	type churnAcc struct {
		commitCount  int
		linesAdded   int
		linesRemoved int
		lastChanged  int64
	}
	type contribKey struct {
		path, email string
	}
	type contribAcc struct {
		author       string
		email        string
		commitCount  int
		linesAdded   int
		linesRemoved int
	}

	churn := map[string]*churnAcc{}
	contrib := map[contribKey]*contribAcc{}

	for _, c := range commits {
		for _, f := range c.Files {
			if churn[f.Path] == nil {
				churn[f.Path] = &churnAcc{}
			}
			ch := churn[f.Path]
			ch.commitCount++
			ch.linesAdded += f.Added
			ch.linesRemoved += f.Removed
			if c.Date > ch.lastChanged {
				ch.lastChanged = c.Date
			}

			ck := contribKey{f.Path, c.Email}
			if contrib[ck] == nil {
				contrib[ck] = &contribAcc{author: c.Author, email: c.Email}
			}
			ca := contrib[ck]
			ca.commitCount++
			ca.linesAdded += f.Added
			ca.linesRemoved += f.Removed
		}
	}

	w := st.NewWriter()
	for path, ch := range churn {
		rec := &gitstore.ChurnRecord{
			Path:         path,
			CommitCount:  ch.commitCount,
			LinesAdded:   ch.linesAdded,
			LinesRemoved: ch.linesRemoved,
			LastChanged:  ch.lastChanged,
		}
		if err := w.PutChurn(path, rec); err != nil {
			return err
		}
	}
	for ck, ca := range contrib {
		rec := &gitstore.ContribRecord{
			Author:       ca.author,
			Email:        ca.email,
			CommitCount:  ca.commitCount,
			LinesAdded:   ca.linesAdded,
			LinesRemoved: ca.linesRemoved,
		}
		if err := w.PutContrib(ck.path, ck.email, rec); err != nil {
			return err
		}
	}
	return w.Flush()
}
