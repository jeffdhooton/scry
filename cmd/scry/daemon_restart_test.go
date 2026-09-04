package main

import (
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/jeffdhooton/scry/internal/rpc"
)

// A daemon restart mid-sweep used to produce one error per transcript the
// sweep was working through. These are the shapes that must be retried, and
// the ones that must not be.
func TestRestartingTellsARestartFromAnAnswer(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection dropped", rpc.ErrConnClosed, true},
		{"dropped and wrapped", fmt.Errorf("get cursor: %w", rpc.ErrConnClosed), true},
		{"refused", syscall.ECONNREFUSED, true},
		{"reset", syscall.ECONNRESET, true},
		{"broken pipe", syscall.EPIPE, true},
		{"badger shutting down", &rpc.Error{Code: -32603, Message: "DB Closed"}, true},
		{"dial refused by text", errors.New(`dial unix /x.sock: connect: connection refused`), true},

		{"unknown method", &rpc.Error{Code: -32601, Message: "unknown method memory.nope"}, false},
		{"bad params", &rpc.Error{Code: -32602, Message: "invalid params"}, false},
		{"a real store error", &rpc.Error{Code: -32603, Message: "key not found"}, false},
		{"anything else", errors.New("no such file or directory"), false},
	}
	for _, c := range cases {
		if got := restarting(c.err); got != c.want {
			t.Errorf("restarting(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}
