package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

type Table struct {
	Name    string
	Columns []Column
	Indexes []Index
}

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

type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}

func ExtractSchema(db *sql.DB) ([]Table, error) {
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

func FormatSchema(tables []Table) string {
	var sb strings.Builder

	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", table.Name))
		sb.WriteString("Columns:\n")

		for _, col := range table.Columns {
			nullable := "NOT NULL"
			if col.IsNullable {
				nullable = "NULL"
			}

			pk := ""
			if col.IsPrimaryKey {
				pk = " (PRIMARY KEY)"
			}

			defaultVal := ""
			if col.DefaultValue.Valid {
				defaultVal = fmt.Sprintf(" DEFAULT %s", col.DefaultValue.String)
			}

			sb.WriteString(fmt.Sprintf("  - %s %s %s%s%s\n",
				col.Name, mapDataType(col), nullable, defaultVal, pk))
		}

		if len(table.Indexes) > 0 {
			sb.WriteString("Indexes:\n")
			for _, idx := range table.Indexes {
				unique := ""
				if idx.IsUnique {
					unique = " (UNIQUE)"
				}
				sb.WriteString(fmt.Sprintf("  - %s on (%s)%s\n",
					idx.Name, strings.Join(idx.Columns, ", "), unique))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func FormatSchemaAsSQL(tables []Table) string {
	var sb strings.Builder

	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("create table %s (\n", table.Name))

		var columnDefs []string
		var primaryKeys []string

		for _, col := range table.Columns {
			var colDef strings.Builder
			colDef.WriteString(fmt.Sprintf("    %s %s", col.Name, strings.ToLower(mapDataType(col))))

			if !col.IsNullable {
				colDef.WriteString(" not null")
			}

			if col.DefaultValue.Valid {
				colDef.WriteString(fmt.Sprintf(" default %s", col.DefaultValue.String))
			}

			columnDefs = append(columnDefs, colDef.String())

			if col.IsPrimaryKey {
				primaryKeys = append(primaryKeys, col.Name)
			}
		}

		sb.WriteString(strings.Join(columnDefs, ",\n"))

		if len(primaryKeys) > 0 {
			sb.WriteString(fmt.Sprintf(",\n    primary key (%s)", strings.Join(primaryKeys, ", ")))
		}

		sb.WriteString("\n);\n\n")

		for _, idx := range table.Indexes {
			unique := ""
			if idx.IsUnique {
				unique = "unique "
			}
			sb.WriteString(fmt.Sprintf("create %sindex %s on %s (%s);\n",
				unique, idx.Name, table.Name, strings.Join(idx.Columns, ", ")))
		}

		if len(table.Indexes) > 0 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func mapDataType(col Column) string {
	switch col.DataType {
	case "character varying":
		if col.CharacterLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", col.CharacterLength.Int64)
		}
		return "VARCHAR(255)"
	case "character", "char":
		if col.CharacterLength.Valid {
			return fmt.Sprintf("CHAR(%d)", col.CharacterLength.Int64)
		}
		return "CHAR"
	case "text":
		return "TEXT"
	case "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "serial":
		return "SERIAL"
	case "bigserial":
		return "BIGSERIAL"
	case "smallserial":
		return "SMALLSERIAL"
	case "boolean":
		return "BOOLEAN"
	case "real":
		return "REAL"
	case "double precision":
		return "DOUBLE PRECISION"
	case "numeric", "decimal":
		if col.NumericPrecision.Valid && col.NumericScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", col.NumericPrecision.Int64, col.NumericScale.Int64)
		} else if col.NumericPrecision.Valid {
			return fmt.Sprintf("DECIMAL(%d)", col.NumericPrecision.Int64)
		}
		return "DECIMAL"
	case "money":
		return "MONEY"
	case "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamp with time zone":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "time without time zone":
		return "TIME"
	case "time with time zone":
		return "TIMETZ"
	case "interval":
		return "INTERVAL"
	case "uuid":
		return "UUID"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	case "xml":
		return "XML"
	case "bytea":
		return "BYTEA"
	case "bit":
		return "BIT"
	case "varbit", "bit varying":
		return "VARBIT"
	case "cidr":
		return "CIDR"
	case "inet":
		return "INET"
	case "macaddr":
		return "MACADDR"
	case "tsvector":
		return "TSVECTOR"
	case "tsquery":
		return "TSQUERY"
	default:
		return strings.ToUpper(col.DataType)
	}
}
