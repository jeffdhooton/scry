package store

// PRIVATE, UNCALLED CONTROL-BOUNDARY HARNESS. It implements no birth tracking,
// observation policy, support oracle, alias generation, adoption or cleanup.

import "errors"

var errIdentityAdmissionScope = errors.New("memory: invalid identity admission scope")

const (
	admissionBody = iota
	admissionFinalizing
	admissionClosed
)

type identityAdmissionOwner struct{ phase int }

// Public maintenance methods open a separate transaction or affect the whole
// DB handle. They can never run through an admission-owned facade, in any
// phase; otherwise outer rollback and closed-facade refusal are bypassed.
func (s *Store) refuseAdmissionMaintenance() error {
	if s.admissionOwner != nil {
		return s.poisonAdmission(errIdentityAdmissionScope)
	}
	return nil
}

func (s *Store) poisonAdmission(err error) error {
	if s.admissionFailure == nil {
		s.admissionFailure = err
	}
	return s.admissionFailure
}

func (s *Store) nestedAdmissionWrite(fn func(*Store) error) (err error) {
	if s.admissionOwner.phase != admissionBody {
		return s.poisonAdmission(errIdentityAdmissionScope)
	}
	defer func() {
		if p := recover(); p != nil {
			s.poisonAdmission(errIdentityAdmissionScope)
			panic(p)
		}
	}()
	err = fn(s)
	if err != nil {
		return s.poisonAdmission(err)
	}
	return s.admissionFailure
}

// Only this outer wrapper can run finalization, after the entire body returns.
// The finalizer argument is a private harness injection point, not a public
// resolver capability or a certificate that a supplied validator is complete.
// Future production admission must bind a reviewed store-owned policy here.
func runIdentityAdmission(st *Store, body, finalizer func(*Store) error) error {
	return runIdentityAdmissionMode(st, body, finalizer, false)
}

// runSerializedIdentityAdmission is a PRIVATE, UNCALLED cooperative writer
// boundary. It is not the fixed production admission policy. Callbacks must
// use only their facade, never synchronously invoke a captured root writer.
// Queue-only root writes remain independent; raw writers and concurrent Close
// are outside the one-genuine-root contract. Finalizer accounting is separate.
func runSerializedIdentityAdmission(st *Store, body, finalizer func(*Store) error) error {
	return runIdentityAdmissionMode(st, body, finalizer, true)
}

func runIdentityAdmissionMode(st *Store, body, finalizer func(*Store) error, serialized bool) error {
	if st == nil {
		return errIdentityAdmissionScope
	}
	if st.txn != nil {
		// Refusing a nested admission entry poisons even an ordinary parent.
		// Catching this error cannot commit its surrounding staged mutations.
		return st.poisonAdmission(errIdentityAdmissionScope)
	}
	if body == nil || finalizer == nil {
		return errIdentityAdmissionScope
	}
	return st.rootAtomicWrite(func(tx *Store) error {
		owner := &identityAdmissionOwner{phase: admissionBody}
		tx.admissionOwner = owner
		defer func() { owner.phase = admissionClosed }()
		if err := body(tx); err != nil {
			return tx.poisonAdmission(err)
		}
		if tx.admissionFailure != nil {
			return tx.admissionFailure
		}
		owner.phase = admissionFinalizing
		// Ordinary update/AtomicWrite calls are frozen during this phase.
		// A future exact finalizer may stage reviewed raw identity/evidence
		// changes through its private primitives; none are implemented here.
		if err := finalizer(tx); err != nil {
			return tx.poisonAdmission(err)
		}
		return tx.admissionFailure
	}, serialized)
}
