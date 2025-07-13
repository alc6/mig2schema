package providers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// ExtractSchemaFromDB extracts schema using SQL queries
func ExtractSchemaFromDB(db *sql.DB) ([]Table, error) {
	slog.Debug("starting schema extraction")
	tables, err := getTables(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	slog.Info("found database tables", "count", len(tables), "tables", tables)

	var schema []Table
	for _, tableName := range tables {
		slog.Debug("processing table", "table", tableName)

		columns, err := getColumns(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		slog.Debug("found table columns", "table", tableName, "count", len(columns))

		indexes, err := getIndexes(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		slog.Debug("found table indexes", "table", tableName, "count", len(indexes))

		schema = append(schema, Table{
			Name:    tableName,
			Columns: columns,
			Indexes: indexes,
		})
	}

	slog.Info("schema extraction completed", "tables", len(schema))
	return schema, nil
}

func getTables(db *sql.DB) ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

func getColumns(db *sql.DB, tableName string) ([]Column, error) {
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			COALESCE(tc.constraint_type = 'PRIMARY KEY', false) as is_primary_key,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu ON 
			c.table_name = kcu.table_name AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc ON 
			kcu.constraint_name = tc.constraint_name AND tc.constraint_type = 'PRIMARY KEY'
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var defaultValue sql.NullString

		if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &defaultValue, &col.IsPrimaryKey, &col.CharacterLength, &col.NumericPrecision, &col.NumericScale); err != nil {
			return nil, err
		}

		col.DefaultValue = defaultValue
		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func getIndexes(db *sql.DB, tableName string) ([]Index, error) {
	query := `
		SELECT 
			i.indexname,
			array_agg(a.attname ORDER BY a.attnum) as columns,
			i.indexdef LIKE '%UNIQUE%' as is_unique
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.tablename
		JOIN pg_index idx ON idx.indexrelid = (
			SELECT oid FROM pg_class WHERE relname = i.indexname
		)
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(idx.indkey)
		WHERE i.tablename = $1 
		AND i.schemaname = 'public'
		AND NOT idx.indisprimary
		GROUP BY i.indexname, i.indexdef
		ORDER BY i.indexname
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var index Index
		var columnsArray string

		if err := rows.Scan(&index.Name, &columnsArray, &index.IsUnique); err != nil {
			return nil, err
		}

		columnsArray = strings.Trim(columnsArray, "{}")
		index.Columns = strings.Split(columnsArray, ",")

		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}