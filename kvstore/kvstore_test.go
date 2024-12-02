package kvstore_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/database"
	"github.com/rohitxdev/go-api-starter/kvstore"
	"github.com/stretchr/testify/assert"
)

func TestKVStore(t *testing.T) {
	var kv *kvstore.Store
	ctx := context.Background()
	dbPath := database.SQLiteDir() + "/test.db"
	t.Run("Create KV store", func(t *testing.T) {
		var err error
		kv, err = kvstore.New(dbPath, time.Second*5)
		assert.NoError(t, err)
	})

	assert.NotNil(t, kv)
	defer func() {
		kv.Close()
		os.RemoveAll(dbPath)
	}()

	t.Run("Set key", func(t *testing.T) {
		assert.Nil(t, kv.Set(ctx, "key", "value"))

		value, err := kv.Get(ctx, "key")
		assert.Equal(t, "value", value)
		assert.NoError(t, err)

		assert.Equal(t, "value", value)
	})

	t.Run("Get key", func(t *testing.T) {
		value, err := kv.Get(ctx, "key")
		assert.NoError(t, err)
		assert.Equal(t, "value", value)
	})

	t.Run("Delete key", func(t *testing.T) {
		//Confirm key exists before deleting it
		value, err := kv.Get(ctx, "key")
		assert.Equal(t, value, "value")
		assert.NotEqual(t, kvstore.ErrKeyNotFound, err)

		assert.Nil(t, kv.Delete(ctx, "key"))

		value, err = kv.Get(ctx, "key")
		assert.Empty(t, value)
		assert.Equal(t, kvstore.ErrKeyNotFound, err)
	})
}
