package storage

import (
	"context"
	"testing"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/testdb?sslmode=disable"

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	// Применить миграции (можно вызвать runMigrations)

	return pool, func() {
		pool.Close()
		container.Terminate(ctx)
	}
}

func TestUserRepo_Create(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &models.User{
		Login:        "testuser",
		PasswordHash: []byte("hash"),
	}
	err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)

	// Проверка дубликата
	err = repo.Create(ctx, user)
	assert.ErrorIs(t, err, storage.ErrUserExists)
}

func TestUserRepo_FindByLogin(t *testing.T) {
	// ...
}
