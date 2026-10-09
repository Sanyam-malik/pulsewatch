package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type groupSQLModel struct {
	bun.BaseModel `bun:"table:groups,alias:g"`
	ID            string    `bun:"id,pk"`
	Name          string    `bun:"name,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type membershipSQLModel struct {
	bun.BaseModel `bun:"table:group_memberships,alias:gm"`
	GroupID       string    `bun:"group_id,pk"`
	UserID        string    `bun:"user_id,pk"`
	Role          string    `bun:"role,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

func toGroupModel(row *groupSQLModel, role string) *Group {
	return &Group{ID: row.ID, Name: row.Name, Role: role, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func (r *SQLRepositoryImpl) CreateFirstOwner(ctx context.Context, user *Model, groupName string) (*Model, *Group, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	count, err := tx.NewSelect().Model((*sqlModel)(nil)).Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	if count != 0 {
		return nil, nil, errors.New("admin already exists")
	}

	now := time.Now().UTC()
	u := &sqlModel{
		ID: uuid.NewString(), Email: user.Email, Password: user.Password, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err = tx.NewInsert().Model(u).Exec(ctx); err != nil {
		return nil, nil, err
	}
	g := new(groupSQLModel)
	err = tx.NewSelect().Model(g).Where("id = ?", LegacyGroupID).Scan(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			return nil, nil, err
		}
		g = &groupSQLModel{ID: uuid.NewString(), Name: strings.TrimSpace(groupName), CreatedAt: now, UpdatedAt: now}
		if _, err = tx.NewInsert().Model(g).Exec(ctx); err != nil {
			return nil, nil, err
		}
	}
	membership := &membershipSQLModel{GroupID: g.ID, UserID: u.ID, Role: RoleOwner, CreatedAt: now, UpdatedAt: now}
	if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	return toDomainModelFromSQL(u), toGroupModel(g, RoleOwner), nil
}

func (r *SQLRepositoryImpl) ResolveMembership(ctx context.Context, userID, groupID string) (*Membership, error) {
	row := new(membershipSQLModel)
	q := r.db.NewSelect().Model(row).Where("user_id = ?", userID)
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	} else {
		q = q.OrderExpr("created_at ASC, group_id ASC").Limit(1)
	}
	if err := q.Scan(ctx); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return &Membership{GroupID: row.GroupID, UserID: row.UserID, Role: row.Role}, nil
}

func (r *SQLRepositoryImpl) ListGroups(ctx context.Context, userID string) ([]Group, error) {
	var rows []groupSQLModel
	err := r.db.NewSelect().Model(&rows).
		Join("JOIN group_memberships AS gm ON gm.group_id = g.id").
		Where("gm.user_id = ?", userID).
		OrderExpr("g.created_at ASC, g.id ASC").Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Group, 0, len(rows))
	for _, row := range rows {
		member, err := r.ResolveMembership(ctx, userID, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *toGroupModel(&row, member.Role))
	}
	return out, nil
}

func (r *SQLRepositoryImpl) CreateGroup(ctx context.Context, userID, name string) (*Group, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	row := &groupSQLModel{ID: uuid.NewString(), Name: strings.TrimSpace(name), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err = tx.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	membership := &membershipSQLModel{GroupID: row.ID, UserID: userID, Role: RoleOwner, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return toGroupModel(row, RoleOwner), nil
}

func (r *SQLRepositoryImpl) ListMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	var rows []struct {
		bun.BaseModel `bun:"table:group_memberships,alias:gm"`
		GroupID       string    `bun:"group_id"`
		UserID        string    `bun:"user_id"`
		Role          string    `bun:"role"`
		CreatedAt     time.Time `bun:"created_at"`
		Email         string    `bun:"email"`
		Active        bool      `bun:"active"`
		UserCreatedAt time.Time `bun:"user_created_at"`
		UserUpdatedAt time.Time `bun:"user_updated_at"`
	}
	err := r.db.NewSelect().TableExpr("group_memberships AS gm").
		ColumnExpr("gm.group_id, gm.user_id, gm.role, gm.created_at").
		ColumnExpr("u.email, u.active, u.created_at AS user_created_at, u.updated_at AS user_updated_at").
		Join("JOIN users AS u ON u.id = gm.user_id").
		Where("gm.group_id = ?", groupID).OrderExpr("gm.created_at ASC, u.email ASC").Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	members := make([]GroupMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, GroupMember{
			User:    &Model{ID: row.UserID, Email: row.Email, Active: row.Active, CreatedAt: row.UserCreatedAt, UpdatedAt: row.UserUpdatedAt},
			GroupID: row.GroupID, Role: row.Role, CreatedAt: row.CreatedAt,
		})
	}
	return members, nil
}

func (r *SQLRepositoryImpl) AddMember(ctx context.Context, groupID, email, password, role string) (*GroupMember, error) {
	if !validRole(role) || role == RoleOwner {
		return nil, errors.New("invalid role")
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	existing := new(sqlModel)
	err = tx.NewSelect().Model(existing).Where("email = ?", email).Scan(ctx)
	if err != nil && !strings.Contains(err.Error(), "no rows") {
		return nil, err
	}
	now := time.Now().UTC()
	user := existing
	if err != nil {
		if password == "" {
			return nil, errors.New("a strong password is required for a new account")
		}
		user = &sqlModel{ID: uuid.NewString(), Email: email, Password: password, Active: true, CreatedAt: now, UpdatedAt: now}
		if _, err = tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return nil, err
		}
	}
	member := &membershipSQLModel{GroupID: groupID, UserID: user.ID, Role: role, CreatedAt: now, UpdatedAt: now}
	if _, err = tx.NewInsert().Model(member).Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &GroupMember{
		User:    &Model{ID: user.ID, Email: user.Email, Active: true, CreatedAt: now, UpdatedAt: now},
		GroupID: groupID, Role: role, CreatedAt: now,
	}, nil
}

func (r *SQLRepositoryImpl) UpdateMemberRole(ctx context.Context, groupID, userID, role string) error {
	if !validRole(role) {
		return errors.New("invalid role")
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	member := new(membershipSQLModel)
	if err = tx.NewSelect().Model(member).Where("group_id = ? AND user_id = ?", groupID, userID).Scan(ctx); err != nil {
		return err
	}
	if member.Role == RoleOwner && role != RoleOwner {
		count, countErr := tx.NewSelect().Model((*membershipSQLModel)(nil)).
			Where("group_id = ? AND role = ?", groupID, RoleOwner).Count(ctx)
		if countErr != nil {
			return countErr
		}
		if count <= 1 {
			return errors.New("cannot remove the last group owner")
		}
	}
	_, err = tx.NewUpdate().Model(member).Set("role = ?", role).
		Set("updated_at = ?", time.Now().UTC()).
		Where("group_id = ? AND user_id = ?", groupID, userID).Exec(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLRepositoryImpl) RemoveMember(ctx context.Context, groupID, userID string) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	member := new(membershipSQLModel)
	err = tx.NewSelect().Model(member).Where("group_id = ? AND user_id = ?", groupID, userID).Scan(ctx)
	if err != nil {
		return err
	}
	if member.Role == RoleOwner {
		count, countErr := tx.NewSelect().Model((*membershipSQLModel)(nil)).
			Where("group_id = ? AND role = ?", groupID, RoleOwner).Count(ctx)
		if countErr != nil {
			return countErr
		}
		if count <= 1 {
			return errors.New("cannot remove the last group owner")
		}
	}
	if _, err = tx.NewDelete().Model(member).Where("group_id = ? AND user_id = ?", groupID, userID).Exec(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLRepositoryImpl) ensureGroupExists(ctx context.Context, groupID string) error {
	exists, err := r.db.NewSelect().Model((*groupSQLModel)(nil)).Where("id = ?", groupID).Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("group not found")
	}
	return nil
}
