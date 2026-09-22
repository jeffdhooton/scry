package assessstore

import (
	"errors"
	"math"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrDispatchLimit = errors.New("assessstore: dispatch limit reached")
	ErrTrialExpired  = errors.New("assessstore: trial expired")
)

// ReserveDispatch commits an attempt before the caller sends any network request.
// Dispatches is a lifetime counter, including attempts whose delivery is unknown;
// neither retention, reopen, nor Resume refunds an attempt. Zero max/until disable
// their respective limits. A job can reserve once; explicit replay is a new job.
func (s *Store) ReserveDispatch(id, owner string, max uint64, until time.Time) error {
	return s.write(func(tx *badger.Txn, st *Status) error {
		var j Job
		if err := get(tx, "j:"+id, &j); err != nil {
			return err
		}
		if j.Status != Running || owner == "" || j.Owner != owner {
			return ErrOwnership
		}
		if !j.DispatchStartedAt.IsZero() {
			return ErrConflict
		}
		if len(j.Packet) == 0 {
			return ErrEvidenceUnavailable
		}
		if st.BlockedReason != "" {
			return ErrBlocked
		}
		now := s.opts.Now()
		if !until.IsZero() && !now.Before(until) {
			return ErrTrialExpired
		}
		if st.Dispatches == math.MaxUint64 || (max != 0 && st.Dispatches >= max) {
			return ErrDispatchLimit
		}
		if now.IsZero() {
			return errors.New("assessstore: dispatch time must be nonzero")
		}
		old := j
		j.DispatchStartedAt = now
		j.UpdatedAt = now
		st.Dispatches++
		// Timestamp growth consumes the completion reservation charged at enqueue.
		return s.saveJob(tx, st, &old, j)
	})
}
