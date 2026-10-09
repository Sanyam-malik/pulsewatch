package monitor_group

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type sqlModel struct {
	bun.BaseModel `bun:"table:monitor_groups,alias:mg"`
	ID            string    `bun:"id,pk"`
	GroupID       string    `bun:"group_id,notnull"`
	Name          string    `bun:"name,notnull"`
	Description   string    `bun:"description,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type sqlMembership struct {
	bun.BaseModel  `bun:"table:monitor_group_monitors,alias:mgm"`
	GroupID        string    `bun:"group_id,pk"`
	MonitorGroupID string    `bun:"monitor_group_id,pk"`
	MonitorID      string    `bun:"monitor_id,pk"`
	CreatedAt      time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type SQLRepository struct {
	db *bun.DB
}

func NewSQLRepository(db *bun.DB) Repository { return &SQLRepository{db: db} }

func (r *SQLRepository) Create(ctx context.Context, entity *Model) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	if err := r.validateMonitors(ctx, groupID, entity.MonitorIDs); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	row := &sqlModel{ID: uuid.NewString(), GroupID: groupID, Name: entity.Name, Description: entity.Description, CreatedAt: now, UpdatedAt: now}
	if _, err = tx.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	for _, monitorID := range entity.MonitorIDs {
		if _, err = tx.NewInsert().Model(&sqlMembership{GroupID: groupID, MonitorGroupID: row.ID, MonitorID: monitorID, CreatedAt: now}).Exec(ctx); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.load(ctx, row, groupID)
}

func (r *SQLRepository) FindByID(ctx context.Context, id string) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	row := new(sqlModel)
	if err := r.db.NewSelect().Model(row).Where("id = ? AND group_id = ?", id, groupID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return r.load(ctx, row, groupID)
}

func (r *SQLRepository) FindByName(ctx context.Context, name string) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	row := new(sqlModel)
	if err := r.db.NewSelect().Model(row).Where("name = ? AND group_id = ?", name, groupID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return r.load(ctx, row, groupID)
}

func (r *SQLRepository) FindAll(ctx context.Context) ([]*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	var rows []sqlModel
	if err := r.db.NewSelect().Model(&rows).Where("group_id = ?", groupID).OrderExpr("name ASC, id ASC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]*Model, 0, len(rows))
	for i := range rows {
		model, err := r.load(ctx, &rows[i], groupID)
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func (r *SQLRepository) Update(ctx context.Context, id string, dto *UpdateDto) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	if dto.MonitorIDs != nil {
		if err := r.validateMonitors(ctx, groupID, *dto.MonitorIDs); err != nil {
			return nil, err
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := tx.NewUpdate().Model((*sqlModel)(nil)).Where("id = ? AND group_id = ?", id, groupID).Set("updated_at = ?", time.Now().UTC())
	if dto.Name != nil {
		query = query.Set("name = ?", *dto.Name)
	}
	if dto.Description != nil {
		query = query.Set("description = ?", *dto.Description)
	}
	if _, err = query.Exec(ctx); err != nil {
		return nil, err
	}
	var exists bool
	if exists, err = tx.NewSelect().Model((*sqlModel)(nil)).Where("id = ? AND group_id = ?", id, groupID).Exists(ctx); err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("monitor group not found")
	}
	if dto.MonitorIDs != nil {
		if _, err = tx.NewDelete().Model((*sqlMembership)(nil)).Where("monitor_group_id = ? AND group_id = ?", id, groupID).Exec(ctx); err != nil {
			return nil, err
		}
		for _, monitorID := range *dto.MonitorIDs {
			if _, err = tx.NewInsert().Model(&sqlMembership{GroupID: groupID, MonitorGroupID: id, MonitorID: monitorID, CreatedAt: time.Now().UTC()}).Exec(ctx); err != nil {
				return nil, err
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *SQLRepository) Delete(ctx context.Context, id string) error {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return errors.New("active group is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.NewDelete().Model((*sqlMembership)(nil)).Where("monitor_group_id = ? AND group_id = ?", id, groupID).Exec(ctx); err != nil {
		return err
	}
	result, err := tx.NewDelete().Model((*sqlModel)(nil)).Where("id = ? AND group_id = ?", id, groupID).Exec(ctx)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("monitor group not found")
	}
	return tx.Commit()
}

func (r *SQLRepository) validateMonitors(ctx context.Context, groupID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := r.db.NewSelect().Table("monitors").Where("id IN (?)", bun.In(ids)).Where("group_id = ?", groupID).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("all monitors must belong to the active group")
	}
	return nil
}

func (r *SQLRepository) load(ctx context.Context, row *sqlModel, groupID string) (*Model, error) {
	var memberships []sqlMembership
	if err := r.db.NewSelect().Model(&memberships).Column("monitor_id").
		Where("monitor_group_id = ? AND group_id = ?", row.ID, groupID).
		OrderExpr("created_at ASC, monitor_id ASC").Scan(ctx); err != nil {
		return nil, err
	}
	ids := make([]string, len(memberships))
	for i, membership := range memberships {
		ids[i] = membership.MonitorID
	}
	return &Model{ID: row.ID, Name: row.Name, Description: row.Description, MonitorIDs: ids, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}
