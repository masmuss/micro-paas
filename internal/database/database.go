package database

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// Register sqlite driver for Bun ORM.
	_ "modernc.org/sqlite"
)

func NewBunDB(dbPath string) (*bun.DB, error) {
	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	return db, nil
}
