package incident

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type sqlIncident struct {
	bun.BaseModel `bun:"table:incidents,alias:i"`
	ID            string     `bun:"id,pk"`
	GroupID       string     `bun:"group_id,notnull"`
	StatusPageID  string     `bun:"status_page_id,notnull"`
	Title         string     `bun:"title,notnull"`
	Status        string     `bun:"status,notnull"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	ResolvedAt    *time.Time `bun:"resolved_at"`
}

type sqlIncidentUpdate struct {
	bun.BaseModel `bun:"table:incident_updates,alias:iu"`
	ID            string    `bun:"id,pk"`
	GroupID       string    `bun:"group_id,notnull"`
	IncidentID    string    `bun:"incident_id,notnull"`
	Status        string    `bun:"status,notnull"`
	Message       string    `bun:"message,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type SQLRepository struct {
	db *bun.DB
}

func NewSQLRepository(db *bun.DB) Repository { return &SQLRepository{db: db} }

func (r *SQLRepository) Create(ctx context.Context, statusPageID, title, status, message string) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	exists, err := r.db.NewSelect().Table("status_pages").Column("id").
		Where("id = ? AND group_id = ?", statusPageID, groupID).Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("status page not found in the active group")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	row := &sqlIncident{ID: uuid.NewString(), GroupID: groupID, StatusPageID: statusPageID, Title: title, Status: status, CreatedAt: now, UpdatedAt: now}
	if status == StatusResolved {
		row.ResolvedAt = &now
	}
	if _, err = tx.NewInsert().Model(row).Exec(ctx); err != nil {
		return nil, err
	}
	update := &sqlIncidentUpdate{ID: uuid.NewString(), GroupID: groupID, IncidentID: row.ID, Status: status, Message: message, CreatedAt: now}
	if _, err = tx.NewInsert().Model(update).Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.load(ctx, row, "", "")
}

func (r *SQLRepository) FindAll(ctx context.Context) ([]*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	var rows []sqlIncident
	if err := r.db.NewSelect().Model(&rows).Where("i.group_id = ?", groupID).
		OrderExpr("i.updated_at DESC, i.id DESC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]*Model, 0, len(rows))
	for i := range rows {
		item, err := r.load(ctx, &rows[i], "", "")
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *SQLRepository) AddUpdate(ctx context.Context, id, status, message string) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok {
		return nil, errors.New("active group is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	row := new(sqlIncident)
	if err = tx.NewSelect().Model(row).Where("i.id = ? AND i.group_id = ?", id, groupID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, errors.New("incident not found")
		}
		return nil, err
	}
	now := time.Now().UTC()
	row.Status, row.UpdatedAt = status, now
	if status == StatusResolved {
		row.ResolvedAt = &now
	} else {
		row.ResolvedAt = nil
	}
	if _, err = tx.NewUpdate().Model(row).Column("status", "updated_at", "resolved_at").
		Where("id = ? AND group_id = ?", id, groupID).Exec(ctx); err != nil {
		return nil, err
	}
	update := &sqlIncidentUpdate{ID: uuid.NewString(), GroupID: groupID, IncidentID: id, Status: status, Message: message, CreatedAt: now}
	if _, err = tx.NewInsert().Model(update).Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.load(ctx, row, "", "")
}

func (r *SQLRepository) FindPublicByStatusPageSlug(ctx context.Context, slug string) ([]*Model, error) {
	var page struct {
		ID    string `bun:"id"`
		Title string `bun:"title"`
		Slug  string `bun:"slug"`
	}
	if err := r.db.NewSelect().Table("status_pages").Column("id", "title", "slug").
		Where("slug = ? AND published = ?", slug, true).Scan(ctx, &page); err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return []*Model{}, nil
		}
		return nil, err
	}
	var rows []sqlIncident
	if err := r.db.NewSelect().Model(&rows).Where("i.status_page_id = ?", page.ID).
		OrderExpr("i.created_at DESC, i.id DESC").Limit(50).Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]*Model, 0, len(rows))
	for i := range rows {
		item, err := r.load(ctx, &rows[i], page.Title, page.Slug)
		if err != nil {
			return nil, err
		}
		item.GroupID = ""
		result = append(result, item)
	}
	return result, nil
}

func (r *SQLRepository) load(ctx context.Context, row *sqlIncident, pageTitle, pageSlug string) (*Model, error) {
	if pageTitle == "" {
		var page struct {
			Title string `bun:"title"`
			Slug  string `bun:"slug"`
		}
		if err := r.db.NewSelect().Table("status_pages").Column("title", "slug").
			Where("id = ? AND group_id = ?", row.StatusPageID, row.GroupID).Scan(ctx, &page); err != nil {
			return nil, err
		}
		pageTitle, pageSlug = page.Title, page.Slug
	}
	var updates []sqlIncidentUpdate
	if err := r.db.NewSelect().Model(&updates).Where("incident_id = ? AND group_id = ?", row.ID, row.GroupID).
		OrderExpr("created_at ASC, id ASC").Scan(ctx); err != nil {
		return nil, err
	}
	result := &Model{
		ID: row.ID, GroupID: row.GroupID, StatusPageID: row.StatusPageID,
		StatusPageTitle: pageTitle, StatusPageSlug: pageSlug, Title: row.Title,
		Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		ResolvedAt: row.ResolvedAt, Updates: make([]Update, len(updates)),
	}
	for i, update := range updates {
		result.Updates[i] = Update{ID: update.ID, Status: update.Status, Message: update.Message, CreatedAt: update.CreatedAt}
	}
	return result, nil
}
