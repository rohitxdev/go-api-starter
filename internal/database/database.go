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

const (
	DirName = ".sqlite"
)

func createDirIfNotExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("Failed to create directory: %w", err)
			}
		} else {
			return fmt.Errorf("Failed to get stats of directory: %w", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

// 'dbName' is the name of the database file. Pass :memory: for in-memory database.
func NewSQLite(dbName string) (*sql.DB, error) {
	if dbName != ":memory:" {
		if err := createDirIfNotExists(DirName); err != nil {
			return nil, err
		}
		dbName = fmt.Sprintf("%s/%s.db", DirName, dbName)
	}

	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		return nil, fmt.Errorf("Failed to open sqlite database: %w", err)
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
		return nil, fmt.Errorf("Failed to ping sqlite database: %w", err)
	}

	return db, nil
}

// 'uri' is the connection string and should be in the form of postgres://user:password@host:port/dbname?sslmode=disable&foo=bar.
func NewPostgreSQL(uri string) (*sql.DB, error) {
	db, err := sql.Open("pgx", uri)
	if err != nil {
		return nil, fmt.Errorf("Failed to open postgres database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping postgres database: %w", err)
	}
	return db, nil
}
