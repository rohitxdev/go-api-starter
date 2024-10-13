package database_test

import (
	"os"
	"testing"

	"github.com/rohitxdev/go-api-starter/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestSqlite(t *testing.T) {
	dbName := "test_db"
	t.Run("Create DB", func(t *testing.T) {
		db, err := database.NewSqlite(dbName)
		assert.Nil(t, err)
		defer db.Close()
	})

	t.Cleanup(func() {
		os.RemoveAll("db")
	})
}
