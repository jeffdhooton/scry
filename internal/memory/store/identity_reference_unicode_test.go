package store

import (
	"bytes"
	"testing"
)

func TestIdentityReferenceExtensionUnicodePairs(t *testing.T) {
	key, raw := referenceBytes(t, referenceFixture())
	for _, extension := range []string{
		`,"extension":"\ud83d\ude00"}`,
		`,"\ud83d\ude00":{"extension":"literal \\ud800"}}`,
		`,"extension":["escaped \" quote","backslash \\","世界 � 😀"],"huge":1e999999}`,
	} {
		candidate := append(bytes.Clone(raw[:len(raw)-1]), extension...)
		if _, err := decodeIdentityReference(key, candidate); err != nil {
			t.Fatal("well-formed opaque extension rejected")
		}
	}
	for _, extension := range []string{
		`,"extension":"\ud800\ud800"}`,
		`,"extension":"\ud800x\udc00"}`,
		`,"extension":"\ud800\\udc00"}`,
		`,"extension":"\udc00\ud800"}`,
		`,"extension":"\ud800\u1234"}`,
	} {
		candidate := append(bytes.Clone(raw[:len(raw)-1]), extension...)
		if _, err := decodeIdentityReference(key, candidate); err == nil {
			t.Fatal("unpaired surrogate accepted")
		}
	}
}
