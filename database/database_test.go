package database_test

import (
	"os"
	"testing"

	"github.com/rohitxdev/go-api-starter/database"
	"github.com/stretchr/testify/assert"
)

func TestSqlite(t *testing.T) {
	dbPath := database.SQLiteDir() + "/test.db"
	t.Run("Create DB", func(t *testing.T) {
		db, err := database.NewSQLite(dbPath)
		assert.Nil(t, err)
		defer func() {
			db.Close()
			os.RemoveAll(dbPath)
		}()
	})
}
