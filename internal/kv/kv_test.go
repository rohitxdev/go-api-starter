package kv_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/internal/kv"
	"github.com/stretchr/testify/assert"
)

func TestKVStore(t *testing.T) {
	var kvstore *kv.Store
	kvName := "test_kv"
	ctx := context.Background()

	t.Run("Create KV store", func(t *testing.T) {
		var err error
		kvstore, err = kv.New(kvName, time.Second*10)
		assert.Nil(t, err)
	})

	assert.NotNil(t, kvstore)

	t.Run("Set key", func(t *testing.T) {
		assert.Nil(t, kvstore.Set(ctx, "key", "value"))

		value, err := kvstore.Get(ctx, "key")
		assert.Equal(t, value, "value")
		assert.Nil(t, err)

		assert.Equal(t, value, "value")
	})

	t.Run("Get key", func(t *testing.T) {
		value, err := kvstore.Get(ctx, "key")
		assert.Nil(t, err)
		assert.Equal(t, value, "value")
	})

	t.Run("Delete key", func(t *testing.T) {
		//Confirm key exists before deleting it
		value, err := kvstore.Get(ctx, "key")
		assert.NotEqual(t, value, "")
		assert.False(t, errors.Is(err, sql.ErrNoRows))

		assert.Nil(t, kvstore.Delete(ctx, "key"))

		value, err = kvstore.Get(ctx, "key")
		assert.Equal(t, value, "")
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Cleanup(func() {
		kvstore.Close()
		os.RemoveAll("db")
	})
}
