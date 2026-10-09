package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"go.uber.org/zap"
)

func newIdentityTestRepository(t *testing.T) *SQLRepositoryImpl {
	t.Helper()
	sqlDB, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL, password TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT TRUE, twofa_secret TEXT,
			twofa_status BOOLEAN NOT NULL DEFAULT FALSE, twofa_last_token TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE groups (
			id TEXT PRIMARY KEY, name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE group_memberships (
			group_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return &SQLRepositoryImpl{db: db}
}

func TestSQLIdentity_FirstOwnerAndMembershipAccess(t *testing.T) {
	repo := newIdentityTestRepository(t)
	ctx := context.Background()
	owner, group, err := repo.CreateFirstOwner(ctx, &Model{
		Email: "owner@example.test", Password: "hash", Active: true,
	}, "Production")
	require.NoError(t, err)
	require.Equal(t, RoleOwner, group.Role)
	require.NotEmpty(t, owner.ID)

	member, err := repo.AddMember(ctx, group.ID, "member@example.test", "hash", RoleViewer)
	require.NoError(t, err)
	require.Equal(t, RoleViewer, member.Role)

	resolved, err := repo.ResolveMembership(ctx, member.User.ID, group.ID)
	require.NoError(t, err)
	require.Equal(t, RoleViewer, resolved.Role)
	members, err := repo.ListMembers(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	require.ErrorContains(t, repo.RemoveMember(ctx, group.ID, owner.ID), "last group owner")
	require.NoError(t, repo.UpdateMemberRole(ctx, group.ID, member.User.ID, RoleMember))
	groups, err := repo.ListGroups(ctx, member.User.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, RoleMember, groups[0].Role)
}

func TestSQLIdentity_BootstrapPreservesExistingUsers(t *testing.T) {
	repo := newIdentityTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_, err := repo.db.Exec(`INSERT INTO users (id, email, password, active, created_at, updated_at)
		VALUES ('legacy-user', 'legacy@example.test', 'hash', TRUE, ?, ?)`, now, now)
	require.NoError(t, err)
	_, _, err = repo.CreateFirstOwner(ctx, &Model{Email: "replacement@example.test", Password: "hash"}, "New group")
	require.ErrorContains(t, err, "admin already exists")
	count, err := repo.FindAllCount(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func TestIdentityService_RoleManagementPolicy(t *testing.T) {
	repo := newIdentityTestRepository(t)
	ctx := context.Background()
	owner, group, err := repo.CreateFirstOwner(ctx, &Model{Email: "owner@example.test", Password: "hash"}, "Production")
	require.NoError(t, err)
	admin, err := repo.AddMember(ctx, group.ID, "admin@example.test", "hash", RoleAdmin)
	require.NoError(t, err)
	viewer, err := repo.AddMember(ctx, group.ID, "viewer@example.test", "hash", RoleViewer)
	require.NoError(t, err)

	service := NewService(repo, nil, zap.NewNop().Sugar())
	_, err = service.AddMember(ctx, admin.User.ID, group.ID, AddMemberDto{Email: "another@example.test", Role: RoleAdmin})
	require.ErrorContains(t, err, "only a group owner")
	require.ErrorContains(t, service.UpdateMemberRole(ctx, admin.User.ID, group.ID, owner.ID, UpdateMemberRoleDto{Role: RoleViewer}), "only a group owner")
	require.ErrorContains(t, service.RemoveMember(ctx, admin.User.ID, group.ID, owner.ID), "only a group owner")
	_, err = service.ListMembers(ctx, viewer.User.ID, group.ID)
	require.ErrorContains(t, err, "administrator access required")

	secondGroup, err := repo.CreateGroup(ctx, owner.ID, "Staging")
	require.NoError(t, err)
	added, err := repo.AddMember(ctx, secondGroup.ID, admin.User.Email, "", RoleMember)
	require.NoError(t, err)
	require.Equal(t, admin.User.ID, added.User.ID)
}
