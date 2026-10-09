package notification_channel

import (
	"context"
	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type sqlModel struct {
	bun.BaseModel `bun:"table:notification_channels,alias:nc"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name,notnull"`
	Type      string    `bun:"type,notnull"`
	Active    bool      `bun:"active,notnull,default:true"`
	IsDefault bool      `bun:"is_default,notnull,default:false"`
	Config    *string   `bun:"config"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	GroupID   string    `bun:"group_id"`
}

func toDomainModelFromSQL(sm *sqlModel) *Model {
	return &Model{
		ID:        sm.ID,
		Name:      sm.Name,
		Type:      sm.Type,
		Active:    sm.Active,
		IsDefault: sm.IsDefault,
		Config:    sm.Config,
		CreatedAt: sm.CreatedAt,
		UpdatedAt: sm.UpdatedAt,
	}
}

func toSQLModel(m *Model) *sqlModel {
	return &sqlModel{
		ID:        m.ID,
		Name:      m.Name,
		Type:      m.Type,
		Active:    m.Active,
		IsDefault: m.IsDefault,
		Config:    m.Config,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type SQLRepositoryImpl struct {
	db *bun.DB
}

func NewSQLRepository(db *bun.DB) Repository {
	return &SQLRepositoryImpl{db: db}
}

func (r *SQLRepositoryImpl) Create(ctx context.Context, entity *Model) (*Model, error) {
	sm := toSQLModel(entity)
	sm.ID = uuid.New().String()
	sm.CreatedAt = time.Now()
	sm.UpdatedAt = time.Now()

	query := r.db.NewInsert().Model(sm).Returning("*")
	if groupID, ok := auth.GroupIDFromContext(ctx); ok {
		sm.GroupID = groupID
	} else {
		query = query.Column("id", "name", "type", "active", "is_default", "config", "created_at", "updated_at")
	}
	_, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return toDomainModelFromSQL(sm), nil
}

func (r *SQLRepositoryImpl) FindByID(ctx context.Context, id string) (*Model, error) {
	sm := new(sqlModel)
	query := r.db.NewSelect().Model(sm).ExcludeColumn("group_id").Where("id = ?", id)
	if groupID, ok := auth.GroupIDFromContext(ctx); ok {
		query = query.Where("group_id = ?", groupID)
	}
	err := query.Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return toDomainModelFromSQL(sm), nil
}

func (r *SQLRepositoryImpl) FindAll(ctx context.Context, page int, limit int, q string) ([]*Model, error) {
	query := r.db.NewSelect().Model((*sqlModel)(nil))
	query = query.ExcludeColumn("group_id")
	if groupID, ok := auth.GroupIDFromContext(ctx); ok {
		query = query.Where("group_id = ?", groupID)
	}

	if q != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+q+"%")
	}

	query = query.Order("created_at DESC").
		Limit(limit).
		Offset(page * limit)

	var sms []*sqlModel
	err := query.Scan(ctx, &sms)
	if err != nil {
		return nil, err
	}

	var models []*Model
	for _, sm := range sms {
		models = append(models, toDomainModelFromSQL(sm))
	}
	return models, nil
}

func (r *SQLRepositoryImpl) UpdateFull(ctx context.Context, id string, entity *Model) error {
	sm := toSQLModel(entity)
	sm.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(sm).
		Where("id = ?", id).
		ExcludeColumn("group_id").
		OmitZero().
		Exec(ctx)
	return err
}

func (r *SQLRepositoryImpl) UpdatePartial(ctx context.Context, id string, entity *UpdateModel) error {
	query := r.db.NewUpdate().Model((*sqlModel)(nil)).Where("id = ?", id)
	if groupID, ok := auth.GroupIDFromContext(ctx); ok {
		query = query.Where("group_id = ?", groupID)
	}

	hasUpdates := false

	if entity.Name != nil {
		query = query.Set("name = ?", *entity.Name)
		hasUpdates = true
	}
	if entity.Type != nil {
		query = query.Set("type = ?", *entity.Type)
		hasUpdates = true
	}
	if entity.Active != nil {
		query = query.Set("active = ?", *entity.Active)
		hasUpdates = true
	}
	if entity.IsDefault != nil {
		query = query.Set("is_default = ?", *entity.IsDefault)
		hasUpdates = true
	}
	if entity.Config != nil {
		query = query.Set("config = ?", *entity.Config)
		hasUpdates = true
	}

	if !hasUpdates {
		return nil
	}

	// Always set updated_at
	query = query.Set("updated_at = ?", time.Now())

	_, err := query.Exec(ctx)
	return err
}

func (r *SQLRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := r.db.NewDelete().Model((*sqlModel)(nil)).Where("id = ?", id)
	if groupID, ok := auth.GroupIDFromContext(ctx); ok {
		query = query.Where("group_id = ?", groupID)
	}
	_, err := query.Exec(ctx)
	return err
}
