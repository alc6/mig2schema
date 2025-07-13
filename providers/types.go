package providers

import "database/sql"

// Table represents a database table with its columns and indexes
type Table struct {
	Name    string
	Columns []Column
	Indexes []Index
}

// Column represents a database column
type Column struct {
	Name              string
	DataType          string
	IsNullable        bool
	DefaultValue      sql.NullString
	IsPrimaryKey      bool
	CharacterLength   sql.NullInt64
	NumericPrecision  sql.NullInt64
	NumericScale      sql.NullInt64
}

// Index represents a database index
type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}