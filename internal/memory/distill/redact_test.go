package distill

import "testing"

import "strings"

func TestRedact(t *testing.T) {
	cases := []struct {
		name            string
		in              string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "anthropic style sk key",
			in:              "here is a key sk-ant-api03-FAKEFAKE1234567890ABCDEFGHIJKL for testing",
			wantContains:    []string{"[REDACTED]"},
			wantNotContains: []string{"FAKEFAKE1234567890ABCDEFGHIJKL"},
		},
		{
			name:            "bearer token",
			in:              "Authorization: Bearer abcdef1234567890ZZZZ.some-token-value",
			wantContains:    []string{"[REDACTED]"},
			wantNotContains: []string{"abcdef1234567890ZZZZ"},
		},
		{
			name: "pem private key block",
			in: "before text\n" +
				"-----BEGIN RSA PRIVATE KEY-----\n" +
				"MIIFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE\n" +
				"MIIFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE\n" +
				"-----END RSA PRIVATE KEY-----\n" +
				"after text",
			wantContains:    []string{"[REDACTED]", "before text", "after text"},
			wantNotContains: []string{"MIIFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE"},
		},
		{
			name:            "github personal access token",
			in:              "token: ghp_FAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE",
			wantContains:    []string{"[REDACTED]"},
			wantNotContains: []string{"ghp_FAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE"},
		},
		{
			name:            "leaves normal prose and short hex strings alone",
			in:              "The commit hash is a3f9c2e and the plan looks well-organized, ship it.",
			wantContains:    []string{"a3f9c2e", "well-organized"},
			wantNotContains: []string{"[REDACTED]"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.in)
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Redact(%q) = %q, want it to contain %q", tc.in, got, want)
				}
			}
			for _, notWant := range tc.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("Redact(%q) = %q, want it to NOT contain %q", tc.in, got, notWant)
				}
			}
		})
	}
}
