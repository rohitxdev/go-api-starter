package kvstore_test

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/pkg/database"
	"github.com/rohitxdev/go-api-starter/pkg/kvstore"
	"github.com/stretchr/testify/assert"
)

func TestKVStore(t *testing.T) {
	var kv *kvstore.KVStore
	kvName := "test_kv"

	t.Run("Create KV store", func(t *testing.T) {
		db, err := database.NewSqlite(kvName)
		assert.Nil(t, err)
		kv, err = kvstore.New(db, time.Second*10)
		assert.Nil(t, err)
	})

	assert.NotNil(t, kv)

	t.Run("Set key", func(t *testing.T) {
		assert.Nil(t, kv.Set("key", "value"))

		value, err := kv.Get("key")
		assert.Equal(t, value, "value")
		assert.Nil(t, err)

		assert.Equal(t, value, "value")
	})

	t.Run("Get key", func(t *testing.T) {
		value, err := kv.Get("key")
		assert.Nil(t, err)
		assert.Equal(t, value, "value")
	})

	t.Run("Delete key", func(t *testing.T) {
		//Confirm key exists before deleting it
		value, err := kv.Get("key")
		assert.NotEqual(t, value, "")
		assert.False(t, errors.Is(err, sql.ErrNoRows))

		assert.Nil(t, kv.Delete("key"))

		value, err = kv.Get("key")
		assert.Equal(t, value, "")
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Cleanup(func() {
		kv.Close()
		os.RemoveAll("db")
	})
}
