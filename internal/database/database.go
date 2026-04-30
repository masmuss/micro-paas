// Package database provides database connection utilities for the application.
package database

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// Register sqlite driver for Bun ORM.
	_ "modernc.org/sqlite"
)

// NewBunDB opens a SQLite database at dbPath and returns a Bun ORM DB instance.
// The returned *bun.DB must be closed by the caller when no longer needed.
// Returns an error if the database cannot be opened.
func NewBunDB(dbPath string) (*bun.DB, error) {
	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", dbPath, err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	return db, nil
}
