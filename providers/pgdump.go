package providers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"strings"
)

// PgDumpProvider uses the pg_dump binary to extract schema
type PgDumpProvider struct{}

// NewPgDumpProvider creates a new pg_dump provider
func NewPgDumpProvider() SchemaProvider {
	return &PgDumpProvider{}
}

// Name returns the provider name
func (p *PgDumpProvider) Name() string {
	return "pg_dump"
}

// IsAvailable checks if pg_dump is available in PATH
func (p *PgDumpProvider) IsAvailable() bool {
	_, err := exec.LookPath("pg_dump")
	return err == nil
}

// ExtractSchema extracts the schema using pg_dump
func (p *PgDumpProvider) ExtractSchema(ctx context.Context, params ExtractParams) (*SchemaResult, error) {
	if params.ConnectionString == "" {
		return nil, fmt.Errorf("pg_dump provider requires connection string")
	}

	// Only SQL format is supported by pg_dump
	if params.Format != FormatSQL {
		return nil, fmt.Errorf("pg_dump provider only supports SQL format")
	}

	slog.Debug("extracting schema using pg_dump provider")

	// Validate connection string format
	_, err := url.Parse(params.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Build pg_dump command
	args := []string{
		"--schema-only",    // Only dump schema, no data
		"--no-owner",       // Don't include ownership information
		"--no-privileges",  // Don't include privilege information
		"--no-tablespaces", // Don't include tablespace information
		"--no-comments",    // Don't include comments
		params.ConnectionString,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("executing pg_dump", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pg_dump failed: %w\nstderr: %s", err, stderr.String())
	}

	rawSQL := stdout.String()

	// Clean up the output
	rawSQL = p.cleanupPgDumpOutput(rawSQL)

	return &SchemaResult{
		RawSQL: rawSQL,
		Format: FormatSQL,
	}, nil
}

// cleanupPgDumpOutput removes unnecessary parts from pg_dump output
func (p *PgDumpProvider) cleanupPgDumpOutput(sql string) string {
	lines := strings.Split(sql, "\n")
	var cleaned []string
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		// Skip PostgreSQL extension-related statements
		if strings.Contains(trimmed, "CREATE EXTENSION") ||
			strings.Contains(trimmed, "COMMENT ON EXTENSION") {
			continue
		}

		// Skip SET statements
		if strings.HasPrefix(trimmed, "SET ") {
			continue
		}

		// Skip SELECT statements (usually for version checks)
		if strings.HasPrefix(trimmed, "SELECT ") {
			continue
		}

		// Skip search_path settings
		if strings.Contains(trimmed, "search_path") {
			skipSection = true
			continue
		}

		// End of search_path section
		if skipSection && strings.HasSuffix(trimmed, ";") {
			skipSection = false
			continue
		}

		if !skipSection {
			cleaned = append(cleaned, line)
		}
	}

	// Join and format
	result := strings.Join(cleaned, "\n")

	// Ensure consistent formatting
	result = strings.ReplaceAll(result, "CREATE TABLE public.", "CREATE TABLE ")
	result = strings.ReplaceAll(result, "ALTER TABLE public.", "ALTER TABLE ")
	result = strings.ReplaceAll(result, "CREATE INDEX", "CREATE INDEX")
	result = strings.ReplaceAll(result, "CREATE UNIQUE INDEX", "CREATE UNIQUE INDEX")

	// Normalize sequences
	result = p.normalizeSequences(result)

	return strings.TrimSpace(result) + "\n"
}

// normalizeSequences converts sequence-related statements to a more readable format
func (p *PgDumpProvider) normalizeSequences(sql string) string {
	// This is a simplified version - in production you might want more sophisticated parsing
	lines := strings.Split(sql, "\n")
	var result []string
	skipSequence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect sequence creation
		if strings.Contains(trimmed, "CREATE SEQUENCE") {
			skipSequence = true
			continue
		}

		// Skip sequence details
		if skipSequence {
			if strings.HasSuffix(trimmed, ";") {
				skipSequence = false
			}
			continue
		}

		// Convert DEFAULT nextval() to simpler form if needed
		if strings.Contains(line, "DEFAULT nextval(") {
			// Keep the line as-is for now, but we could simplify it
			result = append(result, line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
