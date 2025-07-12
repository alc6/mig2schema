package main

import (
	"context"
	"database/sql"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_interfaces.go -package=mocks

// DatabaseManager handles database lifecycle and operations
type DatabaseManager interface {
	// Setup creates and initializes the database connection
	Setup(ctx context.Context) error
	// Close cleans up database resources
	Close(ctx context.Context) error
	// RunMigrations executes the provided migrations
	RunMigrations(migrations []Migration) error
	// GetDB returns the underlying database connection
	GetDB() *sql.DB
}

// SchemaExtractor handles extracting schema information from a database
type SchemaExtractor interface {
	// ExtractSchema retrieves schema information from the database
	ExtractSchema(db *sql.DB) ([]Table, error)
	// FormatSchema formats schema information as human-readable text
	FormatSchema(tables []Table) string
	// FormatSchemaAsSQL formats schema information as SQL CREATE statements
	FormatSchemaAsSQL(tables []Table) string
}

// MigrationReader handles reading migration files
type MigrationReader interface {
	// DiscoverMigrations finds all migration files in the given directory
	DiscoverMigrations(dir string) ([]Migration, error)
}