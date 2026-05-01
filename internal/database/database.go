// Package database provides database connection utilities for the application.
package database

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// Register drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// NewBunDB opens a database connection based on driver and dsn.
func NewBunDB(driver, dsn string) (*bun.DB, error) {
	var sqldb *sql.DB
	var err error
	var db *bun.DB

	switch driver {
	case "sqlite", "sqlite3":
		sqldb, err = sql.Open("sqlite", dsn)
		if err != nil {
			return nil, err
		}
		db = bun.NewDB(sqldb, sqlitedialect.New())
	case "postgres", "postgresql":
		sqldb, err = sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		db = bun.NewDB(sqldb, pgdialect.New())
	case "mysql":
		sqldb, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		db = bun.NewDB(sqldb, mysqldialect.New())
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	return db, nil
}
