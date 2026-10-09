package monitor_group

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

func TestSQLRepositoryScopesMonitorGroupsAndValidatesMembership(t *testing.T) {
	sqlDB, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`
		CREATE TABLE monitors (id TEXT PRIMARY KEY, group_id TEXT NOT NULL);
		CREATE TABLE monitor_groups (
			id TEXT PRIMARY KEY, group_id TEXT NOT NULL, name TEXT NOT NULL,
			description TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
			UNIQUE(group_id, name)
		);
		CREATE TABLE monitor_group_monitors (
			monitor_group_id TEXT NOT NULL, monitor_id TEXT NOT NULL, group_id TEXT NOT NULL,
			created_at DATETIME NOT NULL, PRIMARY KEY(monitor_group_id, monitor_id)
		);
		INSERT INTO monitors (id, group_id) VALUES ('monitor-a', 'group-a'), ('monitor-b', 'group-b');
	`)
	require.NoError(t, err)

	repo := &SQLRepository{db: db}
	ctxA := auth.WithIdentity(context.Background(), "group-a", auth.RoleOwner)
	ctxB := auth.WithIdentity(context.Background(), "group-b", auth.RoleMember)
	_, err = repo.Create(ctxA, &Model{Name: "Cross-group", MonitorIDs: []string{"monitor-b"}})
	require.ErrorContains(t, err, "active group")

	created, err := repo.Create(ctxA, &Model{Name: "Frontend", Description: "Customer-facing", MonitorIDs: []string{"monitor-a"}})
	require.NoError(t, err)
	require.Equal(t, []string{"monitor-a"}, created.MonitorIDs)
	visible, err := repo.FindAll(ctxB)
	require.NoError(t, err)
	require.Empty(t, visible)

	updated, err := repo.Update(ctxA, created.ID, &UpdateDto{MonitorIDs: &[]string{"monitor-b"}})
	require.ErrorContains(t, err, "active group")
	require.Nil(t, updated)
	stillVisible, err := repo.FindByID(ctxA, created.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"monitor-a"}, stillVisible.MonitorIDs)

	require.ErrorContains(t, repo.Delete(ctxB, created.ID), "not found")
	require.NoError(t, repo.Delete(ctxA, created.ID))
}
