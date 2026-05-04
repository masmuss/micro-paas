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

// Factory is a function type that creates a bun.DB instance.
type Factory func(dsn string) (*bun.DB, error)

// factories is a registry of database connection factories.
var factories = map[string]Factory{
	"sqlite":     createSQLite,
	"sqlite3":    createSQLite,
	"postgres":   createPostgres,
	"postgresql": createPostgres,
	"pgx":        createPostgres,
	"mysql":      createMySQL,
}

// NewBunDB opens a database connection based on driver and dsn using the registered factories.
func NewBunDB(driver, dsn string) (*bun.DB, error) {
	factory, ok := factories[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
	return factory(dsn)
}

func createSQLite(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqldb.SetMaxOpenConns(10)
	sqldb.SetMaxIdleConns(5)
	return bun.NewDB(sqldb, sqlitedialect.New()), nil
}

func createPostgres(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(5)
	return bun.NewDB(sqldb, pgdialect.New()), nil
}

func createMySQL(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(5)
	return bun.NewDB(sqldb, mysqldialect.New()), nil
}
