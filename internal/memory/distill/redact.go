package distill

import "regexp"

// redactPatterns is an ordered list of secret shapes to strip before any
// distilled text leaves this package. Order matters only in that each
// pattern is applied to the output of the previous one; the patterns
// themselves target disjoint content so this is not load-bearing today.
var redactPatterns = []*regexp.Regexp{
	// PEM key blocks (private keys of any kind). RE2 has no backreferences,
	// so BEGIN/END labels aren't required to match exactly.
	regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+PRIVATE KEY-----`),
	// Bearer auth headers/tokens, redacted whole (scheme + token).
	regexp.MustCompile(`Bearer\s+[A-Za-z0-9._~+/=-]{10,}`),
	// GitHub personal access / app tokens.
	regexp.MustCompile(`ghp_[A-Za-z0-9]{20,}`),
	// Generic sk-* style API keys (OpenAI, Anthropic, Stripe, etc).
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
}

// Redact strips known secret shapes from text, replacing each match with
// "[REDACTED]". It never errors; unmatched text (including short hex
// strings and normal prose) passes through unchanged.
func Redact(text string) string {
	out := text
	for _, re := range redactPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
