// Package repo provides a wrapper around database.
package repo

import (
	"database/sql"
	"log/slog"
)

type Repo struct {
	db *sql.DB
}

func (repo *Repo) Close() error {
	return repo.db.Close()
}

func New(db *sql.DB) (*Repo, error) {
	r := &Repo{
		db: db,
	}
	if err := r.up(); err != nil {
		return nil, err
	}
	return r, nil
}

func (repo *Repo) up() error {
	if _, err := repo.db.Exec("CREATE EXTENSION IF NOT EXISTS CITEXT;"); err != nil {
		return err
	}

	var exists bool
	if err := repo.db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users');").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		_, err := repo.db.Exec(`
	CREATE TABLE users(
    	id TEXT PRIMARY KEY,
		role TEXT CHECK (role IN ('user', 'admin')) DEFAULT 'user',
    	email CITEXT NOT NULL UNIQUE CHECK (LENGTH(email)<=64),
    	full_name TEXT NOT NULL CHECK (LENGTH(full_name)<=64) DEFAULT '',
    	date_of_birth DATE,
    	gender TEXT CHECK (gender IN ('male', 'female', 'other')),
		phone_number TEXT CHECK (LENGTH(phone_number)<=16),
		account_status TEXT CHECK (account_status IN ('active', 'banned')) DEFAULT 'active',
		image_url TEXT,
		is_verified BOOL DEFAULT FALSE,
    	created_at TIMESTAMPTZ DEFAULT current_timestamp,
    	updated_at TIMESTAMPTZ DEFAULT current_timestamp
	);
	`)
		if err != nil {
			return err
		}
		slog.Info("Created table 'users'")
	}
	return nil
}
