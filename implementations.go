package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/lib/pq"

	"github.com/alc6/mig2schema/providers"
)

type PostgreSQLManager struct {
	container testcontainers.Container
	db        *sql.DB
	connStr   string
}

func NewPostgreSQLManager() DatabaseManager {
	return &PostgreSQLManager{}
}

func (p *PostgreSQLManager) Setup(ctx context.Context) error {
	slog.Debug("starting postgresql container")
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute)),
	)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}
	slog.Debug("got database connection string", "connStr", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.container = container
	p.db = db
	p.connStr = connStr

	slog.Info("postgresql container ready")
	return nil
}

func (p *PostgreSQLManager) Close(ctx context.Context) error {
	if p.db != nil {
		p.db.Close()
	}
	if p.container != nil {
		return p.container.Terminate(ctx)
	}
	return nil
}

func (p *PostgreSQLManager) RunMigrations(migrations []Migration) error {
	for _, migration := range migrations {
		slog.Info("running migration", "name", migration.Name, "file", migration.UpFile)
		
		content, err := os.ReadFile(migration.UpFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migration.UpFile, err)
		}

		if _, err := p.db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
		}
		
		slog.Debug("migration completed successfully", "name", migration.Name)
	}
	slog.Info("all migrations completed successfully", "count", len(migrations))
	return nil
}

func (p *PostgreSQLManager) GetDB() *sql.DB {
	return p.db
}

func (p *PostgreSQLManager) GetConnectionString() string {
	return p.connStr
}

type PostgreSQLSchemaExtractor struct{}

func NewPostgreSQLSchemaExtractor() SchemaExtractor {
	return &PostgreSQLSchemaExtractor{}
}

func (e *PostgreSQLSchemaExtractor) ExtractSchema(db *sql.DB) ([]providers.Table, error) {
	return ExtractSchema(db)
}

func (e *PostgreSQLSchemaExtractor) FormatSchema(tables []providers.Table) string {
	return FormatSchema(tables)
}

func (e *PostgreSQLSchemaExtractor) FormatSchemaAsSQL(tables []providers.Table) string {
	return FormatSchemaAsSQL(tables)
}

type FileMigrationReader struct{}

func NewFileMigrationReader() MigrationReader {
	return &FileMigrationReader{}
}

func (r *FileMigrationReader) DiscoverMigrations(dir string) ([]Migration, error) {
	return ParseMigrations(dir)
}