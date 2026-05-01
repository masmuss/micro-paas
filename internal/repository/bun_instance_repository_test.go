package repository

import (
	"context"
	"database/sql"
	"testing"

	"log/slog"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { sqldb.Close() })
	db := bun.NewDB(sqldb, sqlitedialect.New())
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			container_id TEXT UNIQUE,
			subdomain TEXT UNIQUE NOT NULL,
			port INTEGER DEFAULT 80,
			status TEXT DEFAULT 'running',
			env TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)
	return db
}

func TestBunInstanceRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInstanceRepository(db, slog.Default())
	ctx := context.Background()

	instance := &model.Instance{
		Name:      "test-app",
		Subdomain: "test",
		Port:      8080,
		Status:    model.StatusRunning,
		Env:       map[string]string{"FOO": "BAR"},
	}

	err := repo.Create(ctx, instance)
	require.NoError(t, err)
	assert.Greater(t, instance.ID, int64(0))

	got, err := repo.GetByID(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-app", got.Name)
	assert.Equal(t, "test", got.Subdomain)
	assert.Equal(t, 8080, got.Port)
	assert.Equal(t, map[string]string{"FOO": "BAR"}, got.Env)
}

func TestBunInstanceRepository_GetBySubdomain(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInstanceRepository(db, slog.Default())
	ctx := context.Background()

	err := repo.Create(ctx, &model.Instance{Name: "app1", Subdomain: "sub1", Status: model.StatusRunning})
	require.NoError(t, err)

	got, err := repo.GetBySubdomain(ctx, "sub1")
	require.NoError(t, err)
	assert.Equal(t, "app1", got.Name)
}

func TestBunInstanceRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInstanceRepository(db, slog.Default())
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999)
	assert.ErrorIs(t, err, ErrNotFound)
}
