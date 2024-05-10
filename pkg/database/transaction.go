package database

import (
	"database/sql"
	"fmt"
	"regexp"
)

var schemaNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

func NewTxnWithSchema(db *sql.DB, schema string) (*sql.Tx, error) {
	if !schemaNamePattern.MatchString(schema) {
		return nil, fmt.Errorf("invalid schema name: %s", schema)
	}

	txn, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}

	_, err = txn.Exec(fmt.Sprintf("SET SCHEMA '%s';", schema))
	if err != nil {
		return nil, fmt.Errorf("could not set schema on txn: %w", err)
	}

	return txn, nil
}
