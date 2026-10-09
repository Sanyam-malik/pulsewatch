package tag

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestSQLRepository_UpdateFullIsGroupScoped(t *testing.T) {
	sqlDB, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE tags (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		color TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		group_id TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tags (id, name, color, group_id)
		VALUES ('tag-b', 'Team B', '#000000', 'group-b')`)
	require.NoError(t, err)

	repo := &SQLRepositoryImpl{db: db}
	err = repo.UpdateFull(
		auth.WithIdentity(context.Background(), "group-a", auth.RoleOwner),
		"tag-b",
		&Model{ID: "tag-b", Name: "Changed", Color: "#ffffff"},
	)
	require.NoError(t, err)

	var name string
	require.NoError(t, db.QueryRow("SELECT name FROM tags WHERE id = 'tag-b'").Scan(&name))
	require.Equal(t, "Team B", name)
}
