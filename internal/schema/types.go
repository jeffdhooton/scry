// Package schema defines the core data types for database schema representation.
package schema

// TableRecord is the full descriptor for one database table.
type TableRecord struct {
	Name         string             `json:"name"`
	Schema       string             `json:"schema,omitempty"`
	Type         string             `json:"type"`
	Columns      []ColumnRecord     `json:"columns"`
	Indexes      []IndexRecord      `json:"indexes"`
	ForeignKeys  []ForeignKeyRecord `json:"foreign_keys"`
	ReferencedBy []ForeignKeyRecord `json:"referenced_by"`
	RowEstimate  int64              `json:"row_estimate,omitempty"`
}

type ColumnRecord struct {
	Name         string   `json:"name"`
	OrdinalPos   int      `json:"ordinal_position"`
	DataType     string   `json:"data_type"`
	IsNullable   bool     `json:"is_nullable"`
	DefaultValue *string  `json:"default_value,omitempty"`
	IsPrimaryKey bool     `json:"is_primary_key"`
	IsUnique     bool     `json:"is_unique"`
	Comment      string   `json:"comment,omitempty"`
	EnumValues   []string `json:"enum_values,omitempty"`
}

type IndexRecord struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
	Type      string   `json:"type,omitempty"`
}

type ForeignKeyRecord struct {
	Name             string `json:"name"`
	Table            string `json:"table"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete,omitempty"`
	OnUpdate         string `json:"on_update,omitempty"`
}

type EnumRecord struct {
	Table  string   `json:"table"`
	Column string   `json:"column"`
	Values []string `json:"values"`
}

type SchemaSnapshot struct {
	DBType string        `json:"db_type"`
	DBName string        `json:"db_name"`
	Tables []TableRecord `json:"tables"`
	Enums  []EnumRecord  `json:"enums"`
}
