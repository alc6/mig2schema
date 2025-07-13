package providers

import (
	"fmt"
	"strings"
)

// FormatSchemaInfo formats schema as human-readable text
func FormatSchemaInfo(tables []Table) string {
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

// FormatSchemaSQL formats schema as SQL CREATE statements
func FormatSchemaSQL(tables []Table) string {
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