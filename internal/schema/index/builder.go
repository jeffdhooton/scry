// Package index orchestrates schema introspection and writes results to the
// BadgerDB store.
package index

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/schema"
	"github.com/jeffdhooton/scry/internal/schema/sources/mysql"
	"github.com/jeffdhooton/scry/internal/schema/sources/postgres"
	schemastore "github.com/jeffdhooton/scry/internal/schema/store"
)

type RepoLayout struct {
	ProjectDir   string
	StorageDir   string
	BadgerDir    string
	ManifestPath string
}

type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	ProjectDir    string    `json:"project_dir"`
	DBType        string    `json:"db_type"`
	DBName        string    `json:"db_name"`
	TableCount    int       `json:"table_count"`
	IndexedAt     time.Time `json:"indexed_at"`
	Status        string    `json:"status"`
	ElapsedMs     int64     `json:"elapsed_ms"`
}

func Layout(scryHome, projectDir string) RepoLayout {
	hash := sha256.Sum256([]byte(projectDir))
	short := hex.EncodeToString(hash[:])[:16]
	storage := filepath.Join(scryHome, "repos", short, "schema")
	return RepoLayout{
		ProjectDir:   projectDir,
		StorageDir:   storage,
		BadgerDir:    filepath.Join(storage, "index.db"),
		ManifestPath: filepath.Join(storage, "manifest.json"),
	}
}

func LoadManifest(layout RepoLayout) (*Manifest, error) {
	b, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func Build(scryHome, projectDir, dsn string) (*Manifest, error) {
	dbType, _, err := schema.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	var introspector schema.Introspector
	switch dbType {
	case "mysql":
		introspector = mysql.New()
	case "postgres":
		introspector = postgres.New()
	default:
		return nil, fmt.Errorf("unsupported db type: %s", dbType)
	}

	if err := introspector.Connect(dsn); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer introspector.Close()

	snap, err := introspector.Introspect()
	if err != nil {
		return nil, fmt.Errorf("introspect: %w", err)
	}

	layout := Layout(scryHome, projectDir)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	st, err := schemastore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	start := time.Now()

	if err := WriteSnapshot(st, snap, dsn, projectDir); err != nil {
		return nil, fmt.Errorf("write snapshot: %w", err)
	}

	elapsed := time.Since(start)

	manifest := &Manifest{
		SchemaVersion: schemastore.SchemaVersion,
		ProjectDir:    projectDir,
		DBType:        snap.DBType,
		DBName:        snap.DBName,
		TableCount:    len(snap.Tables),
		IndexedAt:     time.Now(),
		Status:        "ready",
		ElapsedMs:     elapsed.Milliseconds(),
	}

	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(layout.ManifestPath, b, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return manifest, nil
}

// WriteSnapshot resets the store and writes a full schema snapshot.
func WriteSnapshot(st *schemastore.Store, snap *schema.SchemaSnapshot, dsn, projectDir string) error {
	if err := st.Reset(); err != nil {
		return fmt.Errorf("reset store: %w", err)
	}

	if err := st.SetMeta("schema_version", schemastore.SchemaVersion); err != nil {
		return err
	}
	if err := st.SetMeta("dsn", dsn); err != nil {
		return err
	}
	if err := st.SetMeta("db_type", snap.DBType); err != nil {
		return err
	}
	if err := st.SetMeta("db_name", snap.DBName); err != nil {
		return err
	}
	if err := st.SetMeta("project_dir", projectDir); err != nil {
		return err
	}
	if err := st.SetMeta("indexed_at", time.Now()); err != nil {
		return err
	}

	fkRefs := map[string][]schema.ForeignKeyRecord{}
	for _, t := range snap.Tables {
		for _, fk := range t.ForeignKeys {
			fkRefs[fk.ReferencedTable] = append(fkRefs[fk.ReferencedTable], fk)
		}
	}

	w := st.NewWriter()

	for _, t := range snap.Tables {
		t.ReferencedBy = fkRefs[t.Name]
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("marshal table %s: %w", t.Name, err)
		}
		if err := w.PutTable(t.Name, data); err != nil {
			return err
		}

		if err := w.PutName(t.Name, t.Name); err != nil {
			return err
		}
		for _, col := range t.Columns {
			if err := w.PutName(col.Name, t.Name); err != nil {
				return err
			}
		}
	}

	for refTable, fks := range fkRefs {
		for _, fk := range fks {
			data, err := json.Marshal(fk)
			if err != nil {
				return err
			}
			if err := w.PutFKRef(refTable, fk.Table, fk.Column, data); err != nil {
				return err
			}
		}
	}

	for _, e := range snap.Enums {
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if err := w.PutEnum(e.Table, e.Column, data); err != nil {
			return err
		}
	}

	return w.Flush()
}
