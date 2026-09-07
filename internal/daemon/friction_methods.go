package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jeffdhooton/scry/internal/friction"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func (d *Daemon) registerFrictionMethods() {
	for _, method := range []string{"record", "get", "list", "review"} {
		d.server.Register("friction."+method, func(ctx context.Context, raw json.RawMessage) (any, error) {
			return d.handleFriction(ctx, method, raw)
		})
	}
}

type FrictionGetParams struct {
	EventID string `json:"event_id"`
}

func (d *Daemon) handleFriction(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	var operation func(*friction.Store) (any, error)
	switch method {
	case "record":
		var e friction.Event
		if err := friction.Decode(raw, &e); err != nil {
			return nil, invalidParams(err)
		}
		if err := e.Validate(); err != nil {
			return nil, invalidParams(err)
		}
		operation = func(st *friction.Store) (any, error) { return st.Record(e) }
	case "get":
		var p FrictionGetParams
		if err := friction.Decode(raw, &p); err != nil {
			return nil, invalidParams(err)
		}
		if err := friction.ValidateID(p.EventID); err != nil {
			return nil, invalidParams(err)
		}
		operation = func(st *friction.Store) (any, error) { return st.Get(p.EventID) }
	case "list", "review":
		var f friction.Filter
		if err := friction.Decode(raw, &f); err != nil {
			return nil, invalidParams(err)
		}
		if err := f.Validate(); err != nil {
			return nil, invalidParams(err)
		}
		operation = func(st *friction.Store) (any, error) {
			if method == "review" {
				return st.Review(ctx, f)
			}
			return st.List(ctx, f)
		}
	default:
		return nil, fmt.Errorf("unknown friction method")
	}

	// Coordinate opening, operations and closing, including requests still in
	// flight during daemon shutdown. A late request must not reopen the store.
	d.frictionMu.Lock()
	defer d.frictionMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d.frictionClosed {
		return nil, fmt.Errorf("friction journal is closed")
	}
	if d.frictionSt == nil {
		st, err := friction.Open(filepath.Join(d.scryHome(), "friction"))
		if err != nil {
			return nil, err
		}
		d.frictionSt = st
	}
	result, err := operation(d.frictionSt)
	switch {
	case errors.Is(err, friction.ErrInvalid), errors.Is(err, friction.ErrReviewTooLarge):
		return nil, invalidParams(err)
	case errors.Is(err, friction.ErrConflict):
		return nil, &rpc.Error{Code: -32009, Message: err.Error()}
	case errors.Is(err, friction.ErrNotFound):
		return nil, &rpc.Error{Code: -32004, Message: err.Error()}
	default:
		return result, err
	}
}

func (d *Daemon) closeFriction() {
	d.frictionMu.Lock()
	defer d.frictionMu.Unlock()
	d.frictionClosed = true
	if d.frictionSt != nil {
		_ = d.frictionSt.Close()
		d.frictionSt = nil
	}
}
