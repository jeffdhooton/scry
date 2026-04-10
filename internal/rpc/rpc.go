// Package rpc is a tiny JSON-RPC 2.0 implementation over newline-delimited JSON.
//
// Why not net/rpc/jsonrpc: stdlib's jsonrpc is JSON-RPC 1.0, doesn't follow
// the 2.0 envelope, and forces a stateful server type that's awkward to share
// between cobra commands. We only need a small subset of 2.0 — single
// in-flight request per connection, no batch, no notifications — so a hand-
// rolled framing layer is ~150 lines and avoids the impedance mismatch.
//
// Wire format: each request and response is a single JSON object terminated
// by a single '\n'. Both directions speak the same envelope shape.
//
//	Request:  {"jsonrpc":"2.0","id":1,"method":"refs","params":{...}}
//	Response: {"jsonrpc":"2.0","id":1,"result":{...}}
//	Error:    {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"..."}}
package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// JSON-RPC 2.0 standard error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request is the JSON-RPC 2.0 envelope received by the server. Params are
// kept as RawMessage so handlers can unmarshal into their own type.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is the envelope sent back to the client. Exactly one of Result and
// Error is set; never both.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is the JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// HandlerFunc handles one JSON-RPC method. The handler is responsible for
// unmarshaling params into its own type and returning either a result value
// (any JSON-marshalable thing) or an error. Returning a *rpc.Error preserves
// the code; any other error becomes CodeInternalError.
type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// Server is a JSON-RPC 2.0 server bound to a Unix socket (or any net.Listener).
// Methods are registered with Register before calling Serve.
type Server struct {
	mu      sync.RWMutex
	methods map[string]HandlerFunc
}

func NewServer() *Server {
	return &Server{methods: map[string]HandlerFunc{}}
}

func (s *Server) Register(method string, h HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.methods[method] = h
}

// Serve accepts connections from ln and dispatches requests until ctx is done
// or ln returns an error. One goroutine per connection. Each connection
// processes requests sequentially in order so handlers don't need to be
// concurrent-safe (the store handles concurrency itself anyway).
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		go s.serveConn(ctx, conn)
	}
}

func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	// 16 MB max line — should be enough for any single request, including
	// pasted file contents in future hover queries.
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(conn)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = enc.Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: CodeParseError, Message: err.Error()},
			})
			continue
		}
		s.mu.RLock()
		h, ok := s.methods[req.Method]
		s.mu.RUnlock()
		if !ok {
			_ = enc.Encode(Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &Error{Code: CodeMethodNotFound, Message: "method not found: " + req.Method},
			})
			continue
		}
		result, err := h(ctx, req.Params)
		resp := Response{JSONRPC: "2.0", ID: req.ID}
		if err != nil {
			var rpcErr *Error
			if errors.As(err, &rpcErr) {
				resp.Error = rpcErr
			} else {
				resp.Error = &Error{Code: CodeInternalError, Message: err.Error()}
			}
		} else {
			b, mErr := json.Marshal(result)
			if mErr != nil {
				resp.Error = &Error{Code: CodeInternalError, Message: "marshal result: " + mErr.Error()}
			} else {
				resp.Result = b
			}
		}
		if err := enc.Encode(resp); err != nil {
			return // connection broken
		}
	}
}

// Client is a JSON-RPC 2.0 client over a single net.Conn (typically a Unix
// socket). One Client per call site is fine — the cost of opening a Unix
// socket is ~50µs.
type Client struct {
	conn   net.Conn
	enc    *json.Encoder
	dec    *json.Decoder
	nextID atomic.Int64
}

// Dial connects to a Unix domain socket at socketPath.
func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial unix %q: %w", socketPath, err)
	}
	return &Client{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Call sends a request and decodes the response into out. params and out may
// be nil. The error is *rpc.Error if the server returned a JSON-RPC error,
// or a transport error otherwise.
func (c *Client) Call(ctx context.Context, method string, params, out any) error {
	if d, ok := ctx.Deadline(); ok {
		if err := c.conn.SetDeadline(d); err != nil {
			return err
		}
		defer c.conn.SetDeadline(time.Time{})
	}
	id := c.nextID.Add(1)
	idJSON, _ := json.Marshal(id)
	req := Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  method,
	}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		req.Params = b
	}
	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	var resp Response
	if err := c.dec.Decode(&resp); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("daemon closed connection")
		}
		return fmt.Errorf("decode response: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

