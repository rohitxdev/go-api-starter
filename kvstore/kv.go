package kvstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rohitxdev/go-api-starter/database"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExpired  = errors.New("key expired")
)

type Store struct {
	db     *sql.DB
	ticker *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc
}

func New(dbName string, purgeFreq time.Duration) (*Store, error) {
	db, err := database.NewSQLite(dbName)
	if err != nil {
		return nil, fmt.Errorf("Failed to create kv store: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kv_store (
			key TEXT PRIMARY KEY, 
			value TEXT NOT NULL, 
			expires_at TIMESTAMP
		);`); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	store := &Store{
		db:     db,
		ticker: time.NewTicker(purgeFreq),
		ctx:    ctx,
		cancel: cancel,
	}

	go store.cleanUp()

	return store, nil
}

func (kv *Store) cleanUp() {
	for {
		select {
		case <-kv.ticker.C:
			if _, err := kv.db.Exec("DELETE FROM kv_store WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;"); err != nil {
				slog.Error("clean up kv store", slog.Any("error", err))
			}
		case <-kv.ctx.Done():
			return
		}
	}
}

func (kv *Store) Close() error {
	kv.cancel()
	kv.ticker.Stop()

	if err := kv.db.Close(); err != nil {
		return err
	}

	return nil
}

func (kv *Store) Get(ctx context.Context, key string) (string, error) {
	var value string
	var expiresAt sql.NullTime

	err := kv.db.QueryRowContext(ctx, "SELECT value, expires_at FROM kv_store WHERE key = $1", key).Scan(&value, &expiresAt)

	switch {
	case err == sql.ErrNoRows:
		return "", ErrKeyNotFound
	case err != nil:
		return "", err
	case expiresAt.Valid && expiresAt.Time.Before(time.Now()):
		return "", ErrKeyExpired
	}

	return value, nil
}

type setOpts struct {
	expiresIn time.Duration
}

func WithExpiry(expiresIn time.Duration) func(*setOpts) {
	return func(so *setOpts) {
		so.expiresIn = expiresIn
	}
}

func (kv *Store) Set(ctx context.Context, key string, value string, optFuncs ...func(*setOpts)) error {
	opts := setOpts{}
	for _, optFunc := range optFuncs {
		optFunc(&opts)
	}

	var expiresAt sql.NullTime
	if opts.expiresIn > 0 {
		expiresAt = sql.NullTime{Time: time.Now().Add(opts.expiresIn), Valid: true}
	}

	_, err := kv.db.ExecContext(ctx,
		"INSERT INTO kv_store(key, value, expires_at) VALUES($1, $2, $3) "+
			"ON CONFLICT(key) DO UPDATE SET value = $2, expires_at = $3;",
		key, value, expiresAt)

	return err
}

func (kv *Store) Delete(ctx context.Context, key string) error {
	_, err := kv.db.ExecContext(ctx, "DELETE FROM kv_store WHERE key = $1", key)
	return err
}
