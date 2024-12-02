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
	kvName := "test_kv"
	ctx := context.Background()

	t.Run("Create KV store", func(t *testing.T) {
		var err error
		kv, err = kvstore.New(kvName, time.Second*5)
		assert.Nil(t, err)
	})

	assert.NotNil(t, kv)
	defer func() {
		kv.Close()
		os.RemoveAll(database.DirName)
	}()

	t.Run("Set key", func(t *testing.T) {
		assert.Nil(t, kv.Set(ctx, "key", "value"))

		value, err := kv.Get(ctx, "key")
		assert.Equal(t, value, "value")
		assert.Nil(t, err)

		assert.Equal(t, value, "value")
	})

	t.Run("Get key", func(t *testing.T) {
		value, err := kv.Get(ctx, "key")
		assert.Nil(t, err)
		assert.Equal(t, value, "value")
	})

	t.Run("Delete key", func(t *testing.T) {
		//Confirm key exists before deleting it
		value, err := kv.Get(ctx, "key")
		assert.NotEqual(t, value, "")
		assert.NotEqual(t, err, kvstore.ErrKeyNotFound)

		assert.Nil(t, kv.Delete(ctx, "key"))

		value, err = kv.Get(ctx, "key")
		assert.Equal(t, value, "")
		assert.Equal(t, err, kvstore.ErrKeyNotFound)
	})
}