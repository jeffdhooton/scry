// Package postgres provides database introspection for PostgreSQL via pg_catalog.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/jeffdhooton/scry/internal/schema"
)

type Introspector struct {
	conn   *pgx.Conn
	dbName string
}

func New() *Introspector {
	return &Introspector{}
}

func (i *Introspector) Connect(dsn string) error {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}

	var dbName string
	if err := conn.QueryRow(context.Background(), "SELECT current_database()").Scan(&dbName); err != nil {
		_ = conn.Close(context.Background())
		return fmt.Errorf("get database name: %w", err)
	}

	i.conn = conn
	i.dbName = dbName
	return nil
}

func (i *Introspector) Introspect() (*schema.SchemaSnapshot, error) {
	ctx := context.Background()

	tables, err := i.getTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tables: %w", err)
	}

	columns, err := i.getColumns(ctx)
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	indexes, err := i.getIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get indexes: %w", err)
	}

	fks, err := i.getForeignKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("get foreign keys: %w", err)
	}

	pgEnums, err := i.getEnums(ctx)
	if err != nil {
		return nil, fmt.Errorf("get enums: %w", err)
	}

	enumTypes := map[string][]string{}
	for typeName, values := range pgEnums {
		enumTypes[typeName] = values
	}

	var enums []schema.EnumRecord
	for idx := range tables {
		t := &tables[idx]
		t.Columns = columns[t.Name]
		t.Indexes = indexes[t.Name]
		t.ForeignKeys = fks[t.Name]

		for _, ix := range t.Indexes {
			if ix.IsPrimary {
				pkCols := map[string]bool{}
				for _, col := range ix.Columns {
					pkCols[col] = true
				}
				for colIdx := range t.Columns {
					if pkCols[t.Columns[colIdx].Name] {
						t.Columns[colIdx].IsPrimaryKey = true
					}
				}
			}
			if ix.IsUnique && len(ix.Columns) == 1 {
				for colIdx := range t.Columns {
					if t.Columns[colIdx].Name == ix.Columns[0] {
						t.Columns[colIdx].IsUnique = true
					}
				}
			}
		}

		for colIdx := range t.Columns {
			col := &t.Columns[colIdx]
			if vals, ok := enumTypes[col.DataType]; ok {
				col.EnumValues = vals
				enums = append(enums, schema.EnumRecord{
					Table:  t.Name,
					Column: col.Name,
					Values: vals,
				})
			}
		}
	}

	return &schema.SchemaSnapshot{
		DBType: "postgres",
		DBName: i.dbName,
		Tables: tables,
		Enums:  enums,
	}, nil
}

func (i *Introspector) Close() error {
	if i.conn != nil {
		return i.conn.Close(context.Background())
	}
	return nil
}

func (i *Introspector) getTables(ctx context.Context) ([]schema.TableRecord, error) {
	rows, err := i.conn.Query(ctx, `
		SELECT c.relname,
		       CASE c.relkind WHEN 'r' THEN 'BASE TABLE' WHEN 'v' THEN 'VIEW' ELSE 'OTHER' END,
		       GREATEST(c.reltuples::bigint, 0)
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'v')
		ORDER BY c.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []schema.TableRecord
	for rows.Next() {
		var t schema.TableRecord
		if err := rows.Scan(&t.Name, &t.Type, &t.RowEstimate); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (i *Introspector) getColumns(ctx context.Context) (map[string][]schema.ColumnRecord, error) {
	rows, err := i.conn.Query(ctx, `
		SELECT c.relname,
		       a.attname,
		       a.attnum,
		       format_type(a.atttypid, a.atttypmod),
		       NOT a.attnotnull,
		       pg_get_expr(d.adbin, d.adrelid),
		       col_description(a.attrelid, a.attnum),
		       t.typname
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_type t ON t.oid = a.atttypid
		LEFT JOIN pg_catalog.pg_attrdef d ON (a.attrelid = d.adrelid AND a.attnum = d.adnum)
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'v')
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY c.relname, a.attnum`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]schema.ColumnRecord{}
	for rows.Next() {
		var (
			tableName  string
			col        schema.ColumnRecord
			defaultVal *string
			comment    *string
			typeName   string
		)
		if err := rows.Scan(&tableName, &col.Name, &col.OrdinalPos, &col.DataType,
			&col.IsNullable, &defaultVal, &comment, &typeName); err != nil {
			return nil, err
		}
		col.DefaultValue = defaultVal
		if comment != nil {
			col.Comment = *comment
		}
		_ = typeName // used for enum matching in the caller
		result[tableName] = append(result[tableName], col)
	}
	return result, rows.Err()
}

func (i *Introspector) getIndexes(ctx context.Context) (map[string][]schema.IndexRecord, error) {
	rows, err := i.conn.Query(ctx, `
		SELECT ct.relname AS table_name,
		       ci.relname AS index_name,
		       ix.indisunique,
		       ix.indisprimary,
		       array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum))
		FROM pg_catalog.pg_index ix
		JOIN pg_catalog.pg_class ct ON ct.oid = ix.indrelid
		JOIN pg_catalog.pg_class ci ON ci.oid = ix.indexrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = ct.relnamespace
		JOIN pg_catalog.pg_attribute a ON a.attrelid = ix.indrelid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = 'public'
		  AND a.attnum > 0
		GROUP BY ct.relname, ci.relname, ix.indisunique, ix.indisprimary
		ORDER BY ct.relname, ci.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]schema.IndexRecord{}
	for rows.Next() {
		var (
			tableName string
			rec       schema.IndexRecord
			columns   []string
		)
		if err := rows.Scan(&tableName, &rec.Name, &rec.IsUnique, &rec.IsPrimary, &columns); err != nil {
			return nil, err
		}
		rec.Columns = columns
		result[tableName] = append(result[tableName], rec)
	}
	return result, rows.Err()
}

func (i *Introspector) getForeignKeys(ctx context.Context) (map[string][]schema.ForeignKeyRecord, error) {
	rows, err := i.conn.Query(ctx, `
		SELECT ct.relname AS table_name,
		       c.conname,
		       a1.attname AS column_name,
		       ct2.relname AS referenced_table,
		       a2.attname AS referenced_column,
		       c.confdeltype,
		       c.confupdtype
		FROM pg_catalog.pg_constraint c
		JOIN pg_catalog.pg_class ct ON ct.oid = c.conrelid
		JOIN pg_catalog.pg_class ct2 ON ct2.oid = c.confrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = ct.relnamespace
		JOIN pg_catalog.pg_attribute a1 ON a1.attrelid = c.conrelid AND a1.attnum = c.conkey[1]
		JOIN pg_catalog.pg_attribute a2 ON a2.attrelid = c.confrelid AND a2.attnum = c.confkey[1]
		WHERE n.nspname = 'public'
		  AND c.contype = 'f'
		ORDER BY ct.relname, c.conname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]schema.ForeignKeyRecord{}
	for rows.Next() {
		var (
			tableName string
			fk        schema.ForeignKeyRecord
			delType   string
			updType   string
		)
		if err := rows.Scan(&tableName, &fk.Name, &fk.Column,
			&fk.ReferencedTable, &fk.ReferencedColumn,
			&delType, &updType); err != nil {
			return nil, err
		}
		fk.Table = tableName
		fk.OnDelete = pgActionToString(delType)
		fk.OnUpdate = pgActionToString(updType)
		result[tableName] = append(result[tableName], fk)
	}
	return result, rows.Err()
}

func (i *Introspector) getEnums(ctx context.Context) (map[string][]string, error) {
	rows, err := i.conn.Query(ctx, `
		SELECT t.typname, e.enumlabel
		FROM pg_catalog.pg_type t
		JOIN pg_catalog.pg_enum e ON t.oid = e.enumtypid
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
		ORDER BY t.typname, e.enumsortorder`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var typeName, label string
		if err := rows.Scan(&typeName, &label); err != nil {
			return nil, err
		}
		result[typeName] = append(result[typeName], label)
	}
	return result, rows.Err()
}

func pgActionToString(code string) string {
	switch strings.ToLower(code) {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return code
	}
}
