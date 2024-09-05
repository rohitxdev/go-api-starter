package sqlite

import (
	"database/sql"
)

type KV struct {
	db         *sql.DB
	getStmt    *sql.Stmt
	setStmt    *sql.Stmt
	deleteStmt *sql.Stmt
	name       string
}

func NewKV(name string) (*KV, error) {
	db, err := NewDB(name)
	if err != nil {
		return nil, err
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS kv(key TEXT PRIMARY KEY, value TEXT NOT NULL);"); err != nil {
		return nil, err
	}

	getStmt, err := db.Prepare("SELECT value FROM kv WHERE key=$1;")
	if err != nil {
		return nil, err
	}
	setStmt, err := db.Prepare("INSERT INTO kv(key, value) VALUES($1, $2) ON CONFLICT(key) DO UPDATE SET value = $2;")
	if err != nil {
		return nil, err
	}
	deleteStmt, err := db.Prepare("DELETE FROM kv WHERE key = $1")
	if err != nil {
		return nil, err
	}

	return &KV{
		db:         db,
		name:       name,
		getStmt:    getStmt,
		setStmt:    setStmt,
		deleteStmt: deleteStmt,
	}, nil
}

func (kv *KV) Get(key string) (string, error) {
	var value string
	err := kv.getStmt.QueryRow(key).Scan(&value)
	return value, err
}

func (kv *KV) Set(key string, value string) error {
	_, err := kv.setStmt.Exec(key, value)
	return err
}

func (kv *KV) Delete(key string) error {
	_, err := kv.deleteStmt.Exec(key)
	return err
}
