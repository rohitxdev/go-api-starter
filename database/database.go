// Package sqlite provides a wrapper around SQLite database.
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func SQLiteDir() string {
	dir := ".local/sqlite"
	if _, err := os.Stat("go.mod"); err != nil {
		dir = "../" + dir
	}
	return dir
}

// 'dbPath' is the name of the database file. Pass :memory: for in-memory database.
func NewSQLite(dbPath string) (*sql.DB, error) {
	dir := SQLiteDir()
	if dbPath != ":memory:" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	stmts := [...]string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA locking_mode = NORMAL;",
		"PRAGMA busy_timeout = 10000;",
		"PRAGMA cache_size = 10000;",
		"PRAGMA foreign_keys = ON;",
	}

	var errList []error

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return db, nil
}

// 'uri' is the connection string and should be in the form of postgres://user:password@host:port/dbname?sslmode=disable&foo=bar.
func NewPostgreSQL(uri string) (*sql.DB, error) {
	db, err := sql.Open("pgx", uri)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}
	return db, nil
}
