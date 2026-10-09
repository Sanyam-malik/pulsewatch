package incident

import (
	"context"
	"errors"
	"time"

	"github.com/sanyam-malik/pulsewatch/internal/config"
	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoIncident struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	GroupID         primitive.ObjectID `bson:"group_id"`
	StatusPageID    primitive.ObjectID `bson:"status_page_id"`
	StatusPageTitle string             `bson:"status_page_title,omitempty"`
	StatusPageSlug  string             `bson:"status_page_slug,omitempty"`
	Title           string             `bson:"title"`
	Status          string             `bson:"status"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
	ResolvedAt      *time.Time         `bson:"resolved_at,omitempty"`
	Updates         []Update           `bson:"updates"`
}

type MongoRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewMongoRepository(client *mongo.Client, cfg *config.Config) Repository {
	db := client.Database(cfg.DBName)
	return &MongoRepository{db: db, collection: db.Collection("incidents")}
}

func (r *MongoRepository) Create(ctx context.Context, statusPageID, title, status, message string) (*Model, error) {
	groupID, scoped, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, errors.New("active group is required")
	}
	pageID, err := primitive.ObjectIDFromHex(statusPageID)
	if err != nil {
		return nil, errors.New("invalid status page ID")
	}
	var page struct {
		Title string `bson:"title"`
		Slug  string `bson:"slug"`
	}
	if err = r.db.Collection("status_pages").FindOne(ctx, bson.M{"_id": pageID, "group_id": groupID}).Decode(&page); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("status page not found in the active group")
		}
		return nil, err
	}
	now := time.Now().UTC()
	row := &mongoIncident{
		ID: primitive.NewObjectID(), GroupID: groupID, StatusPageID: pageID,
		StatusPageTitle: page.Title, StatusPageSlug: page.Slug,
		Title: title, Status: status, CreatedAt: now, UpdatedAt: now,
		Updates: []Update{{ID: primitive.NewObjectID().Hex(), Status: status, Message: message, CreatedAt: now}},
	}
	if status == StatusResolved {
		row.ResolvedAt = &now
	}
	if _, err = r.collection.InsertOne(ctx, row); err != nil {
		return nil, err
	}
	return toIncident(row), nil
}

func (r *MongoRepository) FindAll(ctx context.Context) ([]*Model, error) {
	groupID, scoped, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, errors.New("active group is required")
	}
	cursor, err := r.collection.Find(ctx, bson.M{"group_id": groupID}, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []mongoIncident
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]*Model, len(rows))
	for i := range rows {
		result[i] = toIncident(&rows[i])
	}
	return result, nil
}

func (r *MongoRepository) AddUpdate(ctx context.Context, id, status, message string) (*Model, error) {
	groupID, scoped, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, errors.New("active group is required")
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid incident ID")
	}
	now := time.Now().UTC()
	var resolvedAt any
	if status == StatusResolved {
		resolvedAt = now
	}
	after := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var row mongoIncident
	err = r.collection.FindOneAndUpdate(ctx, bson.M{"_id": objectID, "group_id": groupID}, bson.M{
		"$set":  bson.M{"status": status, "updated_at": now, "resolved_at": resolvedAt},
		"$push": bson.M{"updates": Update{ID: primitive.NewObjectID().Hex(), Status: status, Message: message, CreatedAt: now}},
	}, after).Decode(&row)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("incident not found")
	}
	if err != nil {
		return nil, err
	}
	return toIncident(&row), nil
}

func (r *MongoRepository) FindPublicByStatusPageSlug(ctx context.Context, slug string) ([]*Model, error) {
	var page struct {
		ID    primitive.ObjectID `bson:"_id"`
		Title string             `bson:"title"`
		Slug  string             `bson:"slug"`
	}
	err := r.db.Collection("status_pages").FindOne(ctx, bson.M{"slug": slug, "published": true}).Decode(&page)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return []*Model{}, nil
	}
	if err != nil {
		return nil, err
	}
	cursor, err := r.collection.Find(ctx, bson.M{"status_page_id": page.ID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []mongoIncident
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]*Model, len(rows))
	for i := range rows {
		rows[i].StatusPageTitle, rows[i].StatusPageSlug = page.Title, page.Slug
		result[i] = toIncident(&rows[i])
		result[i].GroupID = ""
	}
	return result, nil
}

func toIncident(row *mongoIncident) *Model {
	return &Model{
		ID: row.ID.Hex(), GroupID: row.GroupID.Hex(), StatusPageID: row.StatusPageID.Hex(),
		StatusPageTitle: row.StatusPageTitle, StatusPageSlug: row.StatusPageSlug,
		Title: row.Title, Status: row.Status, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, ResolvedAt: row.ResolvedAt, Updates: row.Updates,
	}
}
