package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/repo"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRepo(t *testing.T) {
	ctx := context.Background()

	// Set up the PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpassword",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer postgresC.Terminate(ctx) // nolint:errcheck

	// Get the container host and port
	host, err := postgresC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	port, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	dsn := fmt.Sprintf("postgres://testuser:testpassword@%s:%s/testdb?sslmode=disable", host, port.Port())

	// Connect to PostgreSQL using pgx
	db, err := sql.Open("postgres", dsn)
	assert.Nil(t, err)
	defer db.Close()

	r, err := repo.New(db)
	assert.Nil(t, err)

	t.Run("Create user", func(t *testing.T) {
		id, err := r.CreateUser(ctx, "test@test.com")
		assert.Nil(t, err)
		assert.NotEqual(t, id, "")
	})
}
