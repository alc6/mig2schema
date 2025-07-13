package main

import (
	"database/sql"

	"github.com/alc6/mig2schema/providers"
)


func ExtractSchema(db *sql.DB) ([]providers.Table, error) {
	return providers.ExtractSchemaFromDB(db)
}


func FormatSchema(tables []providers.Table) string {
	return providers.FormatSchemaInfo(tables)
}

func FormatSchemaAsSQL(tables []providers.Table) string {
	return providers.FormatSchemaSQL(tables)
}
