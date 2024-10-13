// Package repo provides a wrapper around database.
package repo

import (
	"database/sql"
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
	_, err := repo.db.Exec(`
	CREATE TABLE IF NOT EXISTS users(
    	id TEXT PRIMARY KEY,
		role TEXT CHECK (role IN ('user', 'admin')) DEFAULT 'user',
    	email CITEXT NOT NULL UNIQUE CHECK (LENGTH(email)<=64),
    	password_hash TEXT NOT NULL CHECK (LENGTH(email)<=72),
		username TEXT UNIQUE CHECK (LENGTH(username)<=32) DEFAULT '',
    	full_name TEXT CHECK (LENGTH(full_name)<=64) DEFAULT '',
    	date_of_birth DATE,
    	gender TEXT CHECK (gender IN ('male', 'female', 'other')),
		phone_number TEXT CHECK (LENGTH(phone_number)<=16),
		account_status TEXT CHECK (account_status IN ('active', 'banned')) DEFAULT 'active',
		image_url TEXT,
    	created_at TIMESTAMPTZ DEFAULT current_timestamp,
    	updated_at TIMESTAMPTZ DEFAULT current_timestamp
	);
	`)
	if err != nil {
		return err
	}
	return err
}
