package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/jeffdhooton/scry/internal/schema"
	schemaindex "github.com/jeffdhooton/scry/internal/schema/index"
	schemastore "github.com/jeffdhooton/scry/internal/schema/store"
)

func (d *Daemon) registerSchemaMethods() {
	d.server.Register("schema.init", d.handleSchemaInit)
	d.server.Register("schema.describe", d.handleSchemaDescribe)
	d.server.Register("schema.relations", d.handleSchemaRelations)
	d.server.Register("schema.search", d.handleSchemaSearch)
	d.server.Register("schema.enums", d.handleSchemaEnums)
	d.server.Register("schema.refresh", d.handleSchemaRefresh)
}

// --- schema.init ---

type SchemaInitParams struct {
	Project   string `json:"project"`
	DSN       string `json:"dsn"`
	DetectEnv bool   `json:"detect_env"`
}

type SchemaInitResult struct {
	Project    string `json:"project"`
	DBType     string `json:"db_type"`
	DBName     string `json:"db_name"`
	TableCount int    `json:"table_count"`
	IndexedAt  string `json:"indexed_at"`
	ElapsedMs  int64  `json:"elapsed_ms"`
}

func (d *Daemon) handleSchemaInit(_ context.Context, raw json.RawMessage) (any, error) {
	var p SchemaInitParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project is required"}
	}
	abs, err := filepath.Abs(p.Project)
	if err != nil {
		return nil, err
	}

	dsn := p.DSN
	if dsn == "" && p.DetectEnv {
		detected, err := schema.DetectDSNFromEnv(abs)
		if err != nil {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "detect DSN: " + err.Error()}
		}
		dsn = detected
	}
	if dsn == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "dsn is required (pass directly or use detect_env:true)"}
	}

	d.schemaRegistry.Evict(abs)

	start := time.Now()
	manifest, err := schemaindex.Build(d.scryHome(), abs, dsn)
	if err != nil {
		return nil, fmt.Errorf("schema index build: %w", err)
	}
	elapsed := time.Since(start)

	layout := schemaindex.Layout(d.scryHome(), abs)
	st, err := schemastore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("reopen schema store after init: %w", err)
	}
	d.schemaRegistry.Put(&SchemaEntry{ProjectDir: abs, Layout: layout, Store: st})

	return &SchemaInitResult{
		Project:    abs,
		DBType:     manifest.DBType,
		DBName:     manifest.DBName,
		TableCount: manifest.TableCount,
		IndexedAt:  manifest.IndexedAt.Format(time.RFC3339),
		ElapsedMs:  elapsed.Milliseconds(),
	}, nil
}

// --- schema.describe ---

type SchemaDescribeParams struct {
	Project string `json:"project"`
	Table   string `json:"table"`
}

func (d *Daemon) handleSchemaDescribe(_ context.Context, raw json.RawMessage) (any, error) {
	var p SchemaDescribeParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" || p.Table == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project and table are required"}
	}
	entry, err := d.schemaRegistry.Get(d.scryHome(), p.Project)
	if err != nil {
		return nil, err
	}
	data, err := entry.Store.GetTable(p.Table)
	if err != nil {
		return nil, err
	}
	var table schema.TableRecord
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, err
	}
	return table, nil
}

// --- schema.relations ---

type SchemaRelationsParams struct {
	Project string `json:"project"`
	Table   string `json:"table"`
}

type SchemaRelationsResult struct {
	Table    string                   `json:"table"`
	Outgoing []schema.ForeignKeyRecord `json:"outgoing"`
	Incoming []schema.ForeignKeyRecord `json:"incoming"`
}

func (d *Daemon) handleSchemaRelations(_ context.Context, raw json.RawMessage) (any, error) {
	var p SchemaRelationsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" || p.Table == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project and table are required"}
	}
	entry, err := d.schemaRegistry.Get(d.scryHome(), p.Project)
	if err != nil {
		return nil, err
	}

	data, err := entry.Store.GetTable(p.Table)
	if err != nil {
		return nil, err
	}
	var table schema.TableRecord
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, err
	}

	outgoing := table.ForeignKeys
	if outgoing == nil {
		outgoing = []schema.ForeignKeyRecord{}
	}

	incomingRaw, err := entry.Store.GetForeignKeysTo(p.Table)
	if err != nil {
		return nil, err
	}
	var incoming []schema.ForeignKeyRecord
	for _, raw := range incomingRaw {
		var fk schema.ForeignKeyRecord
		if err := json.Unmarshal(raw, &fk); err != nil {
			continue
		}
		incoming = append(incoming, fk)
	}
	if incoming == nil {
		incoming = []schema.ForeignKeyRecord{}
	}

	return &SchemaRelationsResult{
		Table:    p.Table,
		Outgoing: outgoing,
		Incoming: incoming,
	}, nil
}

// --- schema.search ---

type SchemaSearchParams struct {
	Project string `json:"project"`
	Query   string `json:"query"`
}

type SchemaSearchResult struct {
	Query   string        `json:"query"`
	Matches []SearchMatch `json:"matches"`
}

type SearchMatch struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Table  string `json:"table,omitempty"`
}

func (d *Daemon) handleSchemaSearch(_ context.Context, raw json.RawMessage) (any, error) {
	var p SchemaSearchParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" || p.Query == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project and query are required"}
	}
	entry, err := d.schemaRegistry.Get(d.scryHome(), p.Project)
	if err != nil {
		return nil, err
	}
	raw2, err := entry.Store.SearchByName(p.Query)
	if err != nil {
		return nil, err
	}

	tables, _ := entry.Store.ListTables()
	tableSet := map[string]bool{}
	for _, t := range tables {
		tableSet[t] = true
	}

	var matches []SearchMatch
	for _, s := range raw2 {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 {
			continue
		}
		token, table := parts[0], parts[1]
		if tableSet[token] {
			matches = append(matches, SearchMatch{Type: "table", Name: token})
		} else {
			matches = append(matches, SearchMatch{Type: "column", Name: token, Table: table})
		}
	}
	if matches == nil {
		matches = []SearchMatch{}
	}

	return &SchemaSearchResult{Query: p.Query, Matches: matches}, nil
}

// --- schema.enums ---

type SchemaEnumsParams struct {
	Project string `json:"project"`
	Table   string `json:"table"`
	Column  string `json:"column"`
}

func (d *Daemon) handleSchemaEnums(_ context.Context, raw json.RawMessage) (any, error) {
	var p SchemaEnumsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project is required"}
	}
	entry, err := d.schemaRegistry.Get(d.scryHome(), p.Project)
	if err != nil {
		return nil, err
	}
	enumsRaw, err := entry.Store.GetEnums(p.Table, p.Column)
	if err != nil {
		return nil, err
	}
	var enums []schema.EnumRecord
	for _, raw := range enumsRaw {
		var e schema.EnumRecord
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		enums = append(enums, e)
	}
	if enums == nil {
		enums = []schema.EnumRecord{}
	}
	return map[string]any{"enums": enums}, nil
}

// --- schema.refresh ---

type SchemaRefreshParams struct {
	Project string `json:"project"`
}

func (d *Daemon) handleSchemaRefresh(ctx context.Context, raw json.RawMessage) (any, error) {
	var p SchemaRefreshParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Project == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "project is required"}
	}
	entry, err := d.schemaRegistry.Get(d.scryHome(), p.Project)
	if err != nil {
		return nil, err
	}

	dsnRaw, err := entry.Store.GetMeta("dsn")
	if err != nil {
		return nil, fmt.Errorf("read stored DSN: %w", err)
	}
	var dsn string
	if err := json.Unmarshal(dsnRaw, &dsn); err != nil {
		return nil, fmt.Errorf("decode stored DSN: %w", err)
	}

	initRaw, _ := json.Marshal(SchemaInitParams{
		Project: p.Project,
		DSN:     dsn,
	})
	return d.handleSchemaInit(ctx, initRaw)
}

// schemaStatusEntries returns the schema index status for all projects.
func (d *Daemon) schemaStatusEntries() []map[string]any {
	var entries []map[string]any
	for _, e := range d.schemaRegistry.Snapshot() {
		entry := map[string]any{"project": e.ProjectDir, "domain": "schema"}
		if m, err := schemaindex.LoadManifest(e.Layout); err == nil {
			entry["indexed_at"] = m.IndexedAt
			entry["db_type"] = m.DBType
			entry["db_name"] = m.DBName
			entry["table_count"] = m.TableCount
		}
		entries = append(entries, entry)
	}

	reposDir := filepath.Join(d.scryHome(), "repos")
	dirs, _ := os.ReadDir(reposDir)
	seen := map[string]bool{}
	for _, e := range d.schemaRegistry.Snapshot() {
		seen[e.ProjectDir] = true
	}
	for _, dir := range dirs {
		schemaManifest := filepath.Join(reposDir, dir.Name(), "schema", "manifest.json")
		if m, err := schemaindex.LoadManifest(schemaindex.RepoLayout{ManifestPath: schemaManifest}); err == nil {
			if !seen[m.ProjectDir] {
				entries = append(entries, map[string]any{
					"project":     m.ProjectDir,
					"domain":      "schema",
					"indexed_at":  m.IndexedAt,
					"db_type":     m.DBType,
					"db_name":     m.DBName,
					"table_count": m.TableCount,
				})
			}
		}
	}
	return entries
}
