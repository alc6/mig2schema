package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alc6/mig2schema/providers"
)

// MockDatabaseManager is a mock implementation of DatabaseManager for testing
type MockDatabaseManager struct {
	SetupFunc            func(ctx context.Context) error
	CloseFunc            func(ctx context.Context) error
	RunMigrationsFunc    func(migrations []Migration) error
	GetDBFunc            func() *sql.DB
	GetConnectionStringFunc func() string
	
	// Track calls for verification
	SetupCalled              bool
	CloseCalled              bool
	RunMigrationsCalled      bool
	GetDBCalled              bool
	GetConnectionStringCalled bool
}

func (m *MockDatabaseManager) Setup(ctx context.Context) error {
	m.SetupCalled = true
	if m.SetupFunc != nil {
		return m.SetupFunc(ctx)
	}
	return nil
}

func (m *MockDatabaseManager) Close(ctx context.Context) error {
	m.CloseCalled = true
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

func (m *MockDatabaseManager) RunMigrations(migrations []Migration) error {
	m.RunMigrationsCalled = true
	if m.RunMigrationsFunc != nil {
		return m.RunMigrationsFunc(migrations)
	}
	return nil
}

func (m *MockDatabaseManager) GetDB() *sql.DB {
	m.GetDBCalled = true
	if m.GetDBFunc != nil {
		return m.GetDBFunc()
	}
	return nil
}

func (m *MockDatabaseManager) GetConnectionString() string {
	m.GetConnectionStringCalled = true
	if m.GetConnectionStringFunc != nil {
		return m.GetConnectionStringFunc()
	}
	return "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
}

// MockSchemaExtractor is a mock implementation of SchemaExtractor for testing
type MockSchemaExtractor struct {
	ExtractSchemaFunc     func(db *sql.DB) ([]providers.Table, error)
	FormatSchemaFunc      func(tables []providers.Table) string
	FormatSchemaAsSQLFunc func(tables []providers.Table) string
}

func (m *MockSchemaExtractor) ExtractSchema(db *sql.DB) ([]providers.Table, error) {
	if m.ExtractSchemaFunc != nil {
		return m.ExtractSchemaFunc(db)
	}
	return []providers.Table{}, nil
}

func (m *MockSchemaExtractor) FormatSchema(tables []providers.Table) string {
	if m.FormatSchemaFunc != nil {
		return m.FormatSchemaFunc(tables)
	}
	return ""
}

func (m *MockSchemaExtractor) FormatSchemaAsSQL(tables []providers.Table) string {
	if m.FormatSchemaAsSQLFunc != nil {
		return m.FormatSchemaAsSQLFunc(tables)
	}
	return ""
}

// MockMigrationReader is a mock implementation of MigrationReader for testing
type MockMigrationReader struct {
	DiscoverMigrationsFunc func(dir string) ([]Migration, error)
}

func (m *MockMigrationReader) DiscoverMigrations(dir string) ([]Migration, error) {
	if m.DiscoverMigrationsFunc != nil {
		return m.DiscoverMigrationsFunc(dir)
	}
	return []Migration{}, nil
}

// TestDatabase is a helper for creating test database instances
type TestDatabase struct {
	*Database
}

// NewTestDatabase creates a test database without requiring Docker
func NewTestDatabase() *TestDatabase {
	return &TestDatabase{
		Database: &Database{
			Container: nil,
			DB:        nil,
			ConnStr:   "test://connection",
		},
	}
}

// SimulateError simulates various database errors for testing
func SimulateError(errType string) error {
	switch errType {
	case "connection":
		return fmt.Errorf("connection refused")
	case "syntax":
		return fmt.Errorf("syntax error at or near 'INVALID'")
	case "permission":
		return fmt.Errorf("permission denied")
	default:
		return fmt.Errorf("simulated error: %s", errType)
	}
}