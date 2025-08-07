package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/alc6/mig2schema/providers"
)

func TestPostgreSQLManager(t *testing.T) {
	t.Run("new_postgresql_manager", func(t *testing.T) {
		manager := NewPostgreSQLManager("postgres:16-alpine")
		assert.NotNil(t, manager)
		var _ DatabaseManager = manager
	})
}

func TestPostgreSQLSchemaExtractor(t *testing.T) {
	t.Run("new_postgresql_schema_extractor", func(t *testing.T) {
		extractor := NewPostgreSQLSchemaExtractor()
		assert.NotNil(t, extractor)
		var _ SchemaExtractor = extractor
	})

	t.Run("delegates_to_functions", func(t *testing.T) {
		extractor := NewPostgreSQLSchemaExtractor()

		tables := []providers.Table{
			{
				Name: "test",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				},
			},
		}

		result := extractor.FormatSchema(tables)
		assert.NotEmpty(t, result)

		sqlResult := extractor.FormatSchemaAsSQL(tables)
		assert.NotEmpty(t, sqlResult)
	})
}

func TestFileMigrationReader(t *testing.T) {
	t.Run("new_file_migration_reader", func(t *testing.T) {
		reader := NewFileMigrationReader()
		assert.NotNil(t, reader)
		var _ MigrationReader = reader
	})
}

func TestImplementationsIntegration(t *testing.T) {
	t.Run("manager_lifecycle", func(t *testing.T) {
		ctx := context.Background()
		manager := NewPostgreSQLManager("postgres:16-alpine").(*PostgreSQLManager)

		err := manager.Setup(ctx)
		assert.NoError(t, err)
		defer manager.Close(ctx)

		assert.NotNil(t, manager.db)
		assert.NotNil(t, manager.container)
		assert.NotEmpty(t, manager.connStr)

		db := manager.GetDB()
		assert.NotNil(t, db)
		assert.Equal(t, manager.db, db)
	})
}
