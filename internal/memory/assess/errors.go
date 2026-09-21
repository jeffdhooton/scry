package assess

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
)

// ErrorCode is an allowlisted diagnostic safe for persistence and status output.
// Provider strings, request contents and transport error messages are never codes.
type ErrorCode string

const (
	CodeUnknown               ErrorCode = "unknown"
	CodeHTTP                  ErrorCode = "http"
	CodeTransport             ErrorCode = "transport"
	CodeDNS                   ErrorCode = "dns"
	CodeConnect               ErrorCode = "connect"
	CodeTLS                   ErrorCode = "tls"
	CodeTimeout               ErrorCode = "timeout"
	CodeCanceled              ErrorCode = "canceled"
	CodeResponseRead          ErrorCode = "response_read"
	CodeResponseSize          ErrorCode = "response_size"
	CodeResponseJSON          ErrorCode = "response_json"
	CodeUsage                 ErrorCode = "usage"
	CodeModelMismatch         ErrorCode = "model_mismatch"
	CodeNoulFields            ErrorCode = "noul_fields"
	CodeAssertionFields       ErrorCode = "assertion_fields"
	CodeAssertionDistribution ErrorCode = "assertion_distribution"
	CodeAssertionConsistency  ErrorCode = "assertion_consistency"
	CodeAnswerCount           ErrorCode = "answer_count"
	CodeMetadata              ErrorCode = "metadata"
	CodeRequestInvalid        ErrorCode = "request_invalid"
)

// Failure holds only an allowlisted category, never an underlying provider error.
// Use FailureCode rather than parsing its text. Only cancellation/deadline
// sentinels are exposed through Unwrap, preserving errors.Is safely.
type Failure struct{ code ErrorCode }

func (e *Failure) Error() string { return "assess: TypeSafe failure (" + string(FailureCode(e)) + ")" }
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.code {
	case CodeCanceled:
		return context.Canceled
	case CodeTimeout:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func failure(code ErrorCode) *Failure { return &Failure{code: allowlistedCode(code)} }

// FailureCode extracts only codes produced by this package. Unknown errors do not
// become status text, even if their messages imitate one of our failures.
// A nil error has no failure code.
func FailureCode(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var f *Failure
	if errors.As(err, &f) && f != nil {
		return allowlistedCode(f.code)
	}
	var he *HTTPError
	if errors.As(err, &he) && he != nil {
		return CodeHTTP
	}
	return CodeUnknown
}

func allowlistedCode(code ErrorCode) ErrorCode {
	switch code {
	case CodeHTTP, CodeTransport, CodeDNS, CodeConnect, CodeTLS, CodeTimeout,
		CodeCanceled, CodeResponseRead, CodeResponseSize, CodeResponseJSON, CodeUsage,
		CodeModelMismatch, CodeNoulFields, CodeAssertionFields, CodeAssertionDistribution,
		CodeAssertionConsistency, CodeAnswerCount, CodeMetadata, CodeRequestInvalid:
		return code
	default:
		return CodeUnknown
	}
}

// Inspect typed causes before discarding them; never infer categories from text.
func requestFailure(ctx context.Context, err error, fallback ErrorCode) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return failure(CodeCanceled)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return failure(CodeTimeout)
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return failure(CodeTimeout)
	}
	// Body-read errors are classified at their boundary unless interrupted above.
	if fallback == CodeResponseRead {
		return failure(fallback)
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return failure(CodeDNS)
	}
	var cert *tls.CertificateVerificationError
	var record tls.RecordHeaderError
	var authority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &cert) || errors.As(err, &record) || errors.As(err, &authority) || errors.As(err, &hostname) || errors.As(err, &invalid) {
		return failure(CodeTLS)
	}
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return failure(CodeConnect)
	}
	return failure(fallback)
}
