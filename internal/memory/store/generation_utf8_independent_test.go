package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestGradeGenerationUTF8EveryInputBoundary(t *testing.T) {
	invalid := []string{"\xff", "\xc0\xaf", "\xe2\x82", "\xed\xa0\x80", "\xf4\x90\x80\x80"}
	for i, suffix := range invalid {
		for _, field := range []string{"birth-episode", "birth-name", "birth-slug", "birth-origin", "load-slug", "load-episode", "alias"} {
			t.Run(fmt.Sprintf("%d/%s", i, field), func(t *testing.T) {
				st := openTemp(t)
				b := gradeGenerationBirth()
				gradeGenerationBirthCommit(t, st, b)
				before := gradeGenerationSnapshot(t, st)
				err := st.AtomicWrite(func(tx *Store) error {
					var refusal error
					if field == "load-slug" {
						_, refusal = loadIdentityGeneration(tx, b.Slug+suffix, "next")
					} else if field == "load-episode" {
						_, refusal = loadIdentityGeneration(tx, b.Slug, "next"+suffix)
					} else if field == "alias" {
						s, err := loadIdentityGeneration(tx, b.Slug, "next")
						if err != nil {
							return err
						}
						_, refusal = s.attest("north-guide" + suffix)
					} else {
						b.Name = "Second Guide"
						b.Slug = "second-guide"
						switch field {
						case "birth-episode":
							b.EpisodeID += suffix
						case "birth-name":
							b.Name += suffix
						case "birth-slug":
							b.Slug += suffix
						case "birth-origin":
							b.Origin += suffix
						}
						_, refusal = beginIdentityGeneration(tx, b)
					}
					if !errors.Is(refusal, errIdentityGeneration) {
						t.Fatalf("invalid input did not explicitly refuse: %v", refusal)
					}
					// Validation failure must precede staging. Deliberately commit
					// this particular validation-only refusal to inspect that fact.
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("invalid text staged persistent state")
				}
			})
		}
	}
}

func TestGradeGenerationUnicodeExactRoundtrip(t *testing.T) {
	for i, text := range []string{"東京-é-�", "e\u0301", "é", "\x00", "\U0001f642", "\u2028\u2029", "\uffff"} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			b.EpisodeID += text
			b.Name += text
			b.Slug = Slugify(b.Name)
			gradeGenerationBirthCommit(t, st, b)
			selectorBefore := gradeGenerationSnapshot(t, st)[identityGenerationPrefix+b.Slug]
			var rec identityGenerationRecord
			if err := json.Unmarshal(selectorBefore, &rec); err != nil {
				t.Fatal(err)
			}
			if rec.Birth.EpisodeID != b.EpisodeID || rec.Birth.Name != b.Name {
				t.Fatal("valid Unicode birth lost exact text")
			}
			alias := Normalize("north-guide" + text)
			id := "next-episode" + text
			gradeGenerationVote(t, st, b.Slug, id, alias, 1)
			gradeGenerationVote(t, st, b.Slug, id, alias, 1)
			state := gradeGenerationSnapshot(t, st)
			if !bytes.Equal(state[identityGenerationPrefix+b.Slug], selectorBefore) {
				t.Fatal("valid Unicode selector changed")
			}
			a := generationAttestation{Generation: rec.ID, Alias: alias, EpisodeID: id}
			var actual generationAttestation
			if err := json.Unmarshal(state[string(generationAttestationKey(a))], &actual); err != nil || actual != a {
				t.Fatal("valid Unicode ledger lost exact alias/episode")
			}
		})
	}
}

func TestGradeGenerationMalformedTextOnDiskRefusesUnchanged(t *testing.T) {
	for _, family := range []string{"selector", "ledger"} {
		for _, malformed := range []string{"raw-byte", "unpaired-surrogate"} {
			t.Run(family+"/"+malformed, func(t *testing.T) {
				st := openTemp(t)
				b := gradeGenerationBirth()
				gradeGenerationBirthCommit(t, st, b)
				gradeGenerationVote(t, st, b.Slug, "vote-1", "north-guide", 1)
				rec, _, _ := identityBirthRecord(b)
				key := identityGenerationPrefix + b.Slug
				target := []byte(b.EpisodeID)
				if family == "ledger" {
					key = string(generationAttestationKey(generationAttestation{Generation: rec.ID, Alias: "north-guide", EpisodeID: "vote-1"}))
					target = []byte("vote-1")
				}
				replacement := []byte{0xff}
				if malformed == "unpaired-surrogate" {
					replacement = []byte(`\ud800`)
				}
				raw := bytes.Replace(gradeGenerationSnapshot(t, st)[key], target, replacement, 1)
				gradeGenerationSet(t, st, key, raw)
				before := gradeGenerationSnapshot(t, st)
				err := st.AtomicWrite(func(tx *Store) error {
					s, err := loadIdentityGeneration(tx, b.Slug, "next")
					if err != nil {
						return err
					}
					_, err = s.attest("north-guide")
					return err
				})
				if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("malformed stored text accepted or rewritten")
				}
			})
		}
	}
}
