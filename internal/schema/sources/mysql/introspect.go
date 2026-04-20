// Package mysql provides database introspection for MySQL via INFORMATION_SCHEMA.
package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/jeffdhooton/scry/internal/schema"
)

type Introspector struct {
	db     *sql.DB
	dbName string
}

func New() *Introspector {
	return &Introspector{}
}

func (i *Introspector) Connect(dsn string) error {
	_, driverDSN, err := schema.ParseDSN(dsn)
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", driverDSN)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping mysql: %w", err)
	}

	var dbName string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&dbName); err != nil {
		_ = db.Close()
		return fmt.Errorf("get database name: %w", err)
	}

	i.db = db
	i.dbName = dbName
	return nil
}

func (i *Introspector) Introspect() (*schema.SchemaSnapshot, error) {
	tables, err := i.getTables()
	if err != nil {
		return nil, fmt.Errorf("get tables: %w", err)
	}

	columns, err := i.getColumns()
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	indexes, err := i.getIndexes()
	if err != nil {
		return nil, fmt.Errorf("get indexes: %w", err)
	}

	fks, err := i.getForeignKeys()
	if err != nil {
		return nil, fmt.Errorf("get foreign keys: %w", err)
	}

	var enums []schema.EnumRecord
	for idx := range tables {
		t := &tables[idx]
		t.Columns = columns[t.Name]
		t.Indexes = indexes[t.Name]
		t.ForeignKeys = fks[t.Name]

		for _, col := range t.Columns {
			if len(col.EnumValues) > 0 {
				enums = append(enums, schema.EnumRecord{
					Table:  t.Name,
					Column: col.Name,
					Values: col.EnumValues,
				})
			}
		}
	}

	return &schema.SchemaSnapshot{
		DBType: "mysql",
		DBName: i.dbName,
		Tables: tables,
		Enums:  enums,
	}, nil
}

func (i *Introspector) Close() error {
	if i.db != nil {
		return i.db.Close()
	}
	return nil
}

func (i *Introspector) getTables() ([]schema.TableRecord, error) {
	rows, err := i.db.Query(`
		SELECT TABLE_NAME, TABLE_TYPE, IFNULL(TABLE_ROWS, 0)
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME`)
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

func (i *Introspector) getColumns() (map[string][]schema.ColumnRecord, error) {
	rows, err := i.db.Query(`
		SELECT TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, COLUMN_TYPE,
		       IS_NULLABLE, COLUMN_DEFAULT, COLUMN_KEY, COLUMN_COMMENT, EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME, ORDINAL_POSITION`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]schema.ColumnRecord{}
	for rows.Next() {
		var (
			tableName  string
			col        schema.ColumnRecord
			nullable   string
			defaultVal sql.NullString
			columnKey  string
			comment    string
			extra      string
			colType    string
		)
		if err := rows.Scan(&tableName, &col.Name, &col.OrdinalPos, &colType,
			&nullable, &defaultVal, &columnKey, &comment, &extra); err != nil {
			return nil, err
		}
		col.DataType = colType
		col.IsNullable = nullable == "YES"
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}
		col.IsPrimaryKey = columnKey == "PRI"
		col.IsUnique = columnKey == "UNI"
		col.Comment = comment
		col.EnumValues = parseEnumValues(colType)

		result[tableName] = append(result[tableName], col)
	}
	return result, rows.Err()
}

func (i *Introspector) getIndexes() (map[string][]schema.IndexRecord, error) {
	rows, err := i.db.Query(`
		SELECT TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX, COLUMN_NAME, NON_UNIQUE, INDEX_TYPE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type indexKey struct {
		table string
		index string
	}
	indexMap := map[indexKey]*schema.IndexRecord{}
	var order []indexKey

	for rows.Next() {
		var (
			tableName, indexName, colName, indexType string
			seqInIndex, nonUnique                   int
		)
		if err := rows.Scan(&tableName, &indexName, &seqInIndex, &colName, &nonUnique, &indexType); err != nil {
			return nil, err
		}
		key := indexKey{tableName, indexName}
		rec, ok := indexMap[key]
		if !ok {
			rec = &schema.IndexRecord{
				Name:      indexName,
				IsUnique:  nonUnique == 0,
				IsPrimary: indexName == "PRIMARY",
				Type:      indexType,
			}
			indexMap[key] = rec
			order = append(order, key)
		}
		rec.Columns = append(rec.Columns, colName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := map[string][]schema.IndexRecord{}
	for _, key := range order {
		result[key.table] = append(result[key.table], *indexMap[key])
	}
	return result, nil
}

func (i *Introspector) getForeignKeys() (map[string][]schema.ForeignKeyRecord, error) {
	rows, err := i.db.Query(`
		SELECT kcu.TABLE_NAME, kcu.CONSTRAINT_NAME, kcu.COLUMN_NAME,
		       kcu.REFERENCED_TABLE_NAME, kcu.REFERENCED_COLUMN_NAME,
		       rc.DELETE_RULE, rc.UPDATE_RULE
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
		  ON rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
		 AND rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		WHERE kcu.TABLE_SCHEMA = DATABASE()
		  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.TABLE_NAME, kcu.CONSTRAINT_NAME`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]schema.ForeignKeyRecord{}
	for rows.Next() {
		var tableName string
		var fk schema.ForeignKeyRecord
		if err := rows.Scan(&tableName, &fk.Name, &fk.Column,
			&fk.ReferencedTable, &fk.ReferencedColumn,
			&fk.OnDelete, &fk.OnUpdate); err != nil {
			return nil, err
		}
		fk.Table = tableName
		result[tableName] = append(result[tableName], fk)
	}
	return result, rows.Err()
}

func parseEnumValues(colType string) []string {
	lower := strings.ToLower(colType)
	var prefix string
	if strings.HasPrefix(lower, "enum(") {
		prefix = "enum("
	} else if strings.HasPrefix(lower, "set(") {
		prefix = "set("
	} else {
		return nil
	}

	inner := colType[len(prefix):]
	if len(inner) > 0 && inner[len(inner)-1] == ')' {
		inner = inner[:len(inner)-1]
	}

	var values []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if !inQuote {
			if ch == '\'' {
				inQuote = true
				current.Reset()
			}
			continue
		}
		if ch == '\'' {
			if i+1 < len(inner) && inner[i+1] == '\'' {
				current.WriteByte('\'')
				i++
			} else {
				values = append(values, current.String())
				inQuote = false
			}
		} else {
			current.WriteByte(ch)
		}
	}
	return values
}
