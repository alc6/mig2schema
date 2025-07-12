package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Database struct {
	Container testcontainers.Container
	DB        *sql.DB
	ConnStr   string
}

func SetupPostgreSQL(ctx context.Context) (*Database, error) {
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
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}
	slog.Debug("got database connection string", "connStr", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("postgresql container ready")
	return &Database{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}, nil
}

func (d *Database) Close(ctx context.Context) error {
	if d.DB != nil {
		d.DB.Close()
	}
	if d.Container != nil {
		return d.Container.Terminate(ctx)
	}
	return nil
}

func (d *Database) RunMigrations(migrations []Migration) error {
	for _, migration := range migrations {
		slog.Debug("running migration", "name", migration.Name, "file", migration.UpFile)

		content, err := os.ReadFile(migration.UpFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migration.UpFile, err)
		}

		if _, err := d.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
		}

		slog.Debug("migration completed successfully", "name", migration.Name)
	}

	slog.Info("all migrations completed successfully", "count", len(migrations))

	return nil
}
