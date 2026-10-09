package incident

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

func TestSQLRepositoryIncidentLifecycleAndGroupScope(t *testing.T) {
	sqlDB, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`
		CREATE TABLE status_pages (id TEXT PRIMARY KEY, slug TEXT, title TEXT, published BOOLEAN, group_id TEXT);
		CREATE TABLE incidents (
			id TEXT PRIMARY KEY, group_id TEXT NOT NULL, status_page_id TEXT NOT NULL, title TEXT NOT NULL,
			status TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, resolved_at DATETIME
		);
		CREATE TABLE incident_updates (
			id TEXT PRIMARY KEY, group_id TEXT NOT NULL, incident_id TEXT NOT NULL, status TEXT NOT NULL,
			message TEXT NOT NULL, created_at DATETIME NOT NULL
		);
		INSERT INTO status_pages (id, slug, title, published, group_id)
			VALUES ('page-a', 'public-a', 'Public A', TRUE, 'group-a'),
			       ('page-b', 'private-b', 'Private B', FALSE, 'group-b');
	`)
	require.NoError(t, err)

	repo := &SQLRepository{db: db}
	ctxA := auth.WithIdentity(context.Background(), "group-a", auth.RoleOwner)
	ctxB := auth.WithIdentity(context.Background(), "group-b", auth.RoleMember)
	created, err := repo.Create(ctxA, "page-a", "API unavailable", StatusInvestigating, "We are investigating.")
	require.NoError(t, err)
	require.Equal(t, StatusInvestigating, created.Status)
	require.Len(t, created.Updates, 1)

	_, err = repo.Create(ctxB, "page-a", "Cross-group", StatusInvestigating, "No access.")
	require.ErrorContains(t, err, "active group")
	require.Empty(t, mustListIncidents(t, repo, ctxB))
	_, err = repo.AddUpdate(ctxB, created.ID, StatusResolved, "Unauthorized.")
	require.ErrorContains(t, err, "not found")

	updated, err := repo.AddUpdate(ctxA, created.ID, StatusIdentified, "A fix is being deployed.")
	require.NoError(t, err)
	require.Equal(t, StatusIdentified, updated.Status)
	require.Len(t, updated.Updates, 2)
	require.Nil(t, updated.ResolvedAt)

	resolved, err := repo.AddUpdate(ctxA, created.ID, StatusResolved, "Service is restored.")
	require.NoError(t, err)
	require.Equal(t, StatusResolved, resolved.Status)
	require.NotNil(t, resolved.ResolvedAt)
	require.Len(t, resolved.Updates, 3)

	public, err := repo.FindPublicByStatusPageSlug(context.Background(), "public-a")
	require.NoError(t, err)
	require.Len(t, public, 1)
	require.Empty(t, public[0].GroupID)
	require.Equal(t, "Public A", public[0].StatusPageTitle)
	require.Equal(t, "public-a", public[0].StatusPageSlug)

	private, err := repo.FindPublicByStatusPageSlug(context.Background(), "private-b")
	require.NoError(t, err)
	require.Empty(t, private)
}

func mustListIncidents(t *testing.T, repo *SQLRepository, ctx context.Context) []*Model {
	t.Helper()
	items, err := repo.FindAll(ctx)
	require.NoError(t, err)
	return items
}
