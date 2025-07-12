package main

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/alc6/mig2schema/providers"
)

func TestFormatSchema(t *testing.T) {
	tables := []providers.Table{
		{
			Name: "users",
			Columns: []providers.Column{
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
			Indexes: []providers.Index{
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
	tables := []providers.Table{
		{
			Name: "products",
			Columns: []providers.Column{
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
			Indexes: []providers.Index{
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


func TestFormatSchemaEmptyTables(t *testing.T) {
	var tables []providers.Table
	result := FormatSchema(tables)
	assert.Empty(t, result)
}

func TestFormatSchemaMultipleTables(t *testing.T) {
	tables := []providers.Table{
		{
			Name: "users",
			Columns: []providers.Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
			},
		},
		{
			Name: "posts",
			Columns: []providers.Column{
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
	testSchema := []providers.Table{
		{
			Name: "users",
			Columns: []providers.Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				{Name: "email", DataType: "character varying", IsNullable: false},
				{Name: "created_at", DataType: "timestamp without time zone", IsNullable: true, 
					DefaultValue: sql.NullString{String: "CURRENT_TIMESTAMP", Valid: true}},
			},
			Indexes: []providers.Index{
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
	tables := []providers.Table{
		{
			Name: "orders",
			Columns: []providers.Column{
				{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				{Name: "user_id", DataType: "integer", IsNullable: false},
				{Name: "status", DataType: "character varying", IsNullable: false},
				{Name: "created_at", DataType: "timestamp without time zone", IsNullable: false},
			},
			Indexes: []providers.Index{
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