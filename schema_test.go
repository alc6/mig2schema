package main

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSchema(t *testing.T) {
	tables := []Table{
		{
			Name: "users",
			Columns: []Column{
				{
					Name:         "id",
					DataType:     "integer",
					IsNullable:   false,
					IsPrimaryKey: true,
				},
				{
					Name:       "email",
					DataType:   "character varying",
					IsNullable: false,
				},
				{
					Name:       "name",
					DataType:   "character varying",
					IsNullable: true,
				},
			},
			Indexes: []Index{
				{
					Name:     "idx_users_email",
					Columns:  []string{"email"},
					IsUnique: true,
				},
			},
		},
	}

	result := FormatSchema(tables)

	assert.Contains(t, result, "Table: users")
	assert.Contains(t, result, "id INTEGER NOT NULL")
	assert.Contains(t, result, "(PRIMARY KEY)")
	assert.Contains(t, result, "email VARCHAR(255) NOT NULL")
	assert.Contains(t, result, "name VARCHAR(255) NULL")
	assert.Contains(t, result, "idx_users_email on (email) (UNIQUE)")
}

func TestFormatSchemaAsSQL(t *testing.T) {
	tables := []Table{
		{
			Name: "products",
			Columns: []Column{
				{
					Name:         "id",
					DataType:     "integer",
					IsNullable:   false,
					IsPrimaryKey: true,
				},
				{
					Name:       "name",
					DataType:   "character varying",
					IsNullable: false,
				},
				{
					Name:       "price",
					DataType:   "numeric",
					IsNullable: true,
				},
			},
			Indexes: []Index{
				{
					Name:     "idx_products_name",
					Columns:  []string{"name"},
					IsUnique: false,
				},
			},
		},
	}

	result := FormatSchemaAsSQL(tables)

	assert.Contains(t, result, "create table products")
	assert.Contains(t, result, "primary key (id)")
	assert.Contains(t, result, "not null")
	assert.Contains(t, result, "create index idx_products_name")
	assert.Contains(t, result, "id integer")
	assert.Contains(t, result, "name varchar(255)")
}

func TestMapDataType(t *testing.T) {
	tests := []struct {
		input    Column
		expected string
	}{
		{Column{DataType: "character varying"}, "VARCHAR(255)"},
		{Column{DataType: "character varying", CharacterLength: sql.NullInt64{Int64: 100, Valid: true}}, "VARCHAR(100)"},
		{Column{DataType: "text"}, "TEXT"},
		{Column{DataType: "integer"}, "INTEGER"},
		{Column{DataType: "serial"}, "SERIAL"},
		{Column{DataType: "bigint"}, "BIGINT"},
		{Column{DataType: "boolean"}, "BOOLEAN"},
		{Column{DataType: "numeric", NumericPrecision: sql.NullInt64{Int64: 10, Valid: true}, NumericScale: sql.NullInt64{Int64: 2, Valid: true}}, "DECIMAL(10,2)"},
		{Column{DataType: "timestamp without time zone"}, "TIMESTAMP"},
		{Column{DataType: "uuid"}, "UUID"},
		{Column{DataType: "json"}, "JSON"},
		{Column{DataType: "unknown_type"}, "UNKNOWN_TYPE"},
	}

	for _, test := range tests {
		result := mapDataType(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestFormatSchemaEmptyTables(t *testing.T) {
	var tables []Table
	result := FormatSchema(tables)
	assert.Empty(t, result)
}

func TestFormatSchemaMultipleTables(t *testing.T) {
	tables := []Table{
		{
			Name: "users",
			Columns: []Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
			},
		},
		{
			Name: "posts",
			Columns: []Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				{Name: "user_id", DataType: "integer", IsNullable: false},
			},
		},
	}

	result := FormatSchema(tables)

	assert.Contains(t, result, "Table: users")
	assert.Contains(t, result, "Table: posts")
}

func TestFormatSchemaWithDefaults(t *testing.T) {
	testSchema := []Table{
		{
			Name: "users",
			Columns: []Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				{Name: "email", DataType: "character varying", IsNullable: false},
				{Name: "created_at", DataType: "timestamp without time zone", IsNullable: true, 
					DefaultValue: sql.NullString{String: "CURRENT_TIMESTAMP", Valid: true}},
			},
			Indexes: []Index{
				{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
			},
		},
	}

	t.Run("format_schema_info_mode_with_defaults", func(t *testing.T) {
		result := FormatSchema(testSchema)
		
		assert.Contains(t, result, "Table: users")
		assert.Contains(t, result, "id INTEGER NOT NULL (PRIMARY KEY)")
		assert.Contains(t, result, "email VARCHAR(255) NOT NULL")
		assert.Contains(t, result, "created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP")
		assert.Contains(t, result, "idx_users_email on (email) (UNIQUE)")
	})

	t.Run("format_schema_sql_mode_with_defaults", func(t *testing.T) {
		result := FormatSchemaAsSQL(testSchema)
		
		assert.Contains(t, result, "create table users")
		assert.Contains(t, result, "id integer not null")
		assert.Contains(t, result, "email varchar(255) not null")
		assert.Contains(t, result, "created_at timestamp default CURRENT_TIMESTAMP")
		assert.Contains(t, result, "primary key (id)")
		assert.Contains(t, result, "create unique index idx_users_email")
	})
}

func TestFormatSchemaComplexIndexes(t *testing.T) {
	tables := []Table{
		{
			Name: "orders",
			Columns: []Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				{Name: "user_id", DataType: "integer", IsNullable: false},
				{Name: "status", DataType: "character varying", IsNullable: false},
				{Name: "created_at", DataType: "timestamp without time zone", IsNullable: false},
			},
			Indexes: []Index{
				{Name: "idx_orders_user_id", Columns: []string{"user_id"}, IsUnique: false},
				{Name: "idx_orders_user_status", Columns: []string{"user_id", "status"}, IsUnique: false},
				{Name: "idx_orders_created", Columns: []string{"created_at"}, IsUnique: false},
			},
		},
	}

	result := FormatSchema(tables)
	assert.Contains(t, result, "idx_orders_user_status on (user_id, status)")
	
	sqlResult := FormatSchemaAsSQL(tables)
	assert.Contains(t, sqlResult, "create index idx_orders_user_status on orders (user_id, status)")
}