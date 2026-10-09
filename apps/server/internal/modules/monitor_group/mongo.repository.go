package monitor_group

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sanyam-malik/pulsewatch/internal/config"
	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoModel struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	GroupID     primitive.ObjectID `bson:"group_id"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	MonitorIDs  []string           `bson:"monitor_ids"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

type MongoRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewMongoRepository(client *mongo.Client, cfg *config.Config) Repository {
	db := client.Database(cfg.DBName)
	collection := db.Collection("monitor_groups")
	if _, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		panic("Failed to create monitor group index: " + err.Error())
	}
	return &MongoRepository{db: db, collection: collection}
}

func (r *MongoRepository) Create(ctx context.Context, entity *Model) (*Model, error) {
	groupID, scoped, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, errors.New("active group is required")
	}
	if err := r.validateMonitors(ctx, groupID, entity.MonitorIDs); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := &mongoModel{ID: primitive.NewObjectID(), GroupID: groupID, Name: entity.Name, Description: entity.Description, MonitorIDs: entity.MonitorIDs, CreatedAt: now, UpdatedAt: now}
	if _, err := r.collection.InsertOne(ctx, row); err != nil {
		return nil, err
	}
	return toModel(row), nil
}

func (r *MongoRepository) FindByID(ctx context.Context, id string) (*Model, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	filter, err := r.groupFilter(ctx, bson.M{"_id": objectID})
	if err != nil {
		return nil, err
	}
	var row mongoModel
	err = r.collection.FindOne(ctx, filter).Decode(&row)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toModel(&row), nil
}

func (r *MongoRepository) FindByName(ctx context.Context, name string) (*Model, error) {
	filter, err := r.groupFilter(ctx, bson.M{"name": name})
	if err != nil {
		return nil, err
	}
	var row mongoModel
	err = r.collection.FindOne(ctx, filter).Decode(&row)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toModel(&row), nil
}

func (r *MongoRepository) FindAll(ctx context.Context) ([]*Model, error) {
	filter, err := r.groupFilter(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []mongoModel
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]*Model, len(rows))
	for i := range rows {
		result[i] = toModel(&rows[i])
	}
	return result, nil
}

func (r *MongoRepository) Update(ctx context.Context, id string, dto *UpdateDto) (*Model, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	filter, err := r.groupFilter(ctx, bson.M{"_id": objectID})
	if err != nil {
		return nil, err
	}
	groupID, _, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if dto.MonitorIDs != nil {
		if err := r.validateMonitors(ctx, groupID, *dto.MonitorIDs); err != nil {
			return nil, err
		}
	}
	set := bson.M{"updated_at": time.Now().UTC()}
	if dto.Name != nil {
		set["name"] = *dto.Name
	}
	if dto.Description != nil {
		set["description"] = *dto.Description
	}
	if dto.MonitorIDs != nil {
		set["monitor_ids"] = *dto.MonitorIDs
	}
	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, errors.New("monitor group not found")
	}
	return r.FindByID(ctx, id)
}

func (r *MongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	filter, err := r.groupFilter(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err == nil && result.DeletedCount == 0 {
		return errors.New("monitor group not found")
	}
	return err
}

func (r *MongoRepository) validateMonitors(ctx context.Context, groupID primitive.ObjectID, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	objectIDs := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		parsed, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid monitor ID %q", id)
		}
		objectIDs[i] = parsed
	}
	count, err := r.db.Collection("monitor").CountDocuments(ctx, bson.M{
		"_id": bson.M{"$in": objectIDs}, "group_id": groupID,
	})
	if err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errors.New("all monitors must belong to the active group")
	}
	return nil
}

func (r *MongoRepository) groupFilter(ctx context.Context, filter bson.M) (bson.M, error) {
	groupID, scoped, err := auth.MongoGroupIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !scoped {
		return nil, errors.New("active group is required")
	}
	filter["group_id"] = groupID
	return filter, nil
}

func toModel(row *mongoModel) *Model {
	ids := row.MonitorIDs
	if ids == nil {
		ids = []string{}
	}
	return &Model{ID: row.ID.Hex(), Name: row.Name, Description: strings.TrimSpace(row.Description), MonitorIDs: ids, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
