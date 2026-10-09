package monitor_tag

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

func TestSQLRepository_RejectsCrossGroupAssociation(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`CREATE TABLE monitors (id TEXT PRIMARY KEY, group_id TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE tags (id TEXT PRIMARY KEY, group_id TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE monitor_tags (id TEXT PRIMARY KEY, monitor_id TEXT NOT NULL,
		tag_id TEXT NOT NULL, created_at DATETIME, updated_at DATETIME)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO monitors (id, group_id) VALUES ('monitor-a', 'group-a')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tags (id, group_id) VALUES ('tag-b', 'group-b')`)
	require.NoError(t, err)

	repo := &SQLRepositoryImpl{db: db}
	_, err = repo.Create(auth.WithIdentity(context.Background(), "group-a", auth.RoleOwner),
		&Model{MonitorID: "monitor-a", TagID: "tag-b"})
	require.ErrorContains(t, err, "must belong to the selected group")
}
