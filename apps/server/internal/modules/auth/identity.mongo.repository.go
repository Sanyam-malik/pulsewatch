package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoGroup struct {
	ID         primitive.ObjectID `bson:"_id"`
	Name       string             `bson:"name"`
	OwnerCount int64              `bson:"owner_count"`
	CreatedAt  time.Time          `bson:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt"`
}

type mongoMembership struct {
	GroupID   primitive.ObjectID `bson:"group_id"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Role      string             `bson:"role"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}

func (r *RepositoryImpl) groups() *mongo.Collection {
	return r.db.Collection("groups")
}

func (r *RepositoryImpl) memberships() *mongo.Collection {
	return r.db.Collection("group_memberships")
}

func (r *RepositoryImpl) CreateFirstOwner(ctx context.Context, user *Model, groupName string) (*Model, *Group, error) {
	lockID := "first-owner"
	now := time.Now().UTC()
	_, err := r.db.Collection("identity_bootstrap").InsertOne(ctx, bson.M{
		"_id": lockID, "locked_until": now.Add(time.Minute),
	})
	if err != nil {
		// A stale lock from a crashed startup may be reclaimed, but an active
		// first-owner transaction must never race another bootstrap request.
		_, _ = r.db.Collection("identity_bootstrap").DeleteOne(ctx, bson.M{
			"_id": lockID, "locked_until": bson.M{"$lte": now},
		})
		_, err = r.db.Collection("identity_bootstrap").InsertOne(ctx, bson.M{
			"_id": lockID, "locked_until": now.Add(time.Minute),
		})
		if err != nil {
			return nil, nil, errors.New("admin already exists or bootstrap is in progress")
		}
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = r.db.Collection("identity_bootstrap").DeleteOne(ctx, bson.M{"_id": lockID})
		}
	}()
	count, err := r.FindAllCount(ctx)
	if err != nil {
		return nil, nil, err
	}
	if count != 0 {
		return nil, nil, errors.New("admin already exists")
	}
	createdUser, err := r.Create(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	groupRow := mongoGroup{ID: primitive.NewObjectID(), Name: strings.TrimSpace(groupName), OwnerCount: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err = r.groups().InsertOne(ctx, groupRow); err != nil {
		oid, _ := primitive.ObjectIDFromHex(createdUser.ID)
		_, _ = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
		return nil, nil, err
	}
	oid, err := primitive.ObjectIDFromHex(createdUser.ID)
	if err != nil {
		_, _ = r.groups().DeleteOne(ctx, bson.M{"_id": groupRow.ID})
		_, _ = r.collection.DeleteOne(ctx, bson.M{"_id": mustObjectID(createdUser.ID)})
		return nil, nil, err
	}
	_, err = r.memberships().InsertOne(ctx, mongoMembership{GroupID: groupRow.ID, UserID: oid, Role: RoleOwner, CreatedAt: groupRow.CreatedAt, UpdatedAt: groupRow.UpdatedAt})
	if err != nil {
		_, _ = r.groups().DeleteOne(ctx, bson.M{"_id": groupRow.ID})
		_, _ = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
		return nil, nil, err
	}
	committed = true
	return createdUser, &Group{ID: groupRow.ID.Hex(), Name: groupRow.Name, Role: RoleOwner, CreatedAt: groupRow.CreatedAt, UpdatedAt: groupRow.UpdatedAt}, nil
}

func mustObjectID(id string) primitive.ObjectID {
	oid, _ := primitive.ObjectIDFromHex(id)
	return oid
}

func (r *RepositoryImpl) ResolveMembership(ctx context.Context, userID, groupID string) (*Membership, error) {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	var member mongoMembership
	filter := bson.M{"user_id": userOID}
	if groupID != "" {
		groupOID, parseErr := primitive.ObjectIDFromHex(groupID)
		if parseErr != nil {
			return nil, nil
		}
		filter["group_id"] = groupOID
		err = r.memberships().FindOne(ctx, filter).Decode(&member)
	} else {
		err = r.memberships().FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: 1}})).Decode(&member)
		if errors.Is(err, mongo.ErrNoDocuments) {
			// A lazy legacy backfill supports pre-migration MongoDB deployments.
			// Only the earliest legacy account receives owner privileges.
			var user mongoModel
			if err = r.collection.FindOne(ctx, bson.M{"_id": userOID}).Decode(&user); err != nil {
				return nil, err
			}
			legacyID, _ := primitive.ObjectIDFromHex("000000000000000000000001")
			_, err = r.groups().UpdateOne(ctx, bson.M{"_id": legacyID}, bson.M{"$setOnInsert": bson.M{
				"name": "Default group", "owner_count": int64(0), "createdAt": user.CreatedAt, "updatedAt": time.Now().UTC(),
			}}, options.Update().SetUpsert(true))
			if err != nil {
				return nil, err
			}
			for _, collectionName := range []string{"monitor", "proxies", "notification_channel", "maintenance", "status_pages", "tags", "api_keys"} {
				if _, err = r.db.Collection(collectionName).UpdateMany(ctx, bson.M{"group_id": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"group_id": legacyID}}); err != nil {
					return nil, err
				}
			}
			var earliest mongoModel
			err = r.collection.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}})).Decode(&earliest)
			if err != nil {
				return nil, err
			}
			role := RoleViewer
			if earliest.ID == userOID {
				role = RoleOwner
			}
			member = mongoMembership{GroupID: legacyID, UserID: userOID, Role: role, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
			result, updateErr := r.memberships().UpdateOne(ctx,
				bson.M{"group_id": legacyID, "user_id": userOID},
				bson.M{"$setOnInsert": member}, options.Update().SetUpsert(true))
			if updateErr != nil {
				return nil, updateErr
			}
			if role == RoleOwner && result.UpsertedCount == 1 {
				if _, err = r.groups().UpdateOne(ctx, bson.M{"_id": legacyID}, bson.M{"$inc": bson.M{"owner_count": int64(1)}}); err != nil {
					return nil, err
				}
			}
			if err != nil {
				return nil, err
			}
			return &Membership{GroupID: legacyID.Hex(), UserID: userID, Role: role}, nil
		}
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Membership{GroupID: member.GroupID.Hex(), UserID: userID, Role: member.Role}, nil
}

func (r *RepositoryImpl) ListGroups(ctx context.Context, userID string) ([]Group, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.memberships().Find(ctx, bson.M{"user_id": oid}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []mongoMembership
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]Group, 0, len(rows))
	for _, row := range rows {
		var g mongoGroup
		if err = r.groups().FindOne(ctx, bson.M{"_id": row.GroupID}).Decode(&g); err != nil {
			return nil, err
		}
		out = append(out, Group{ID: g.ID.Hex(), Name: g.Name, Role: row.Role, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt})
	}
	return out, nil
}

func (r *RepositoryImpl) CreateGroup(ctx context.Context, userID, name string) (*Group, error) {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	g := mongoGroup{ID: primitive.NewObjectID(), Name: strings.TrimSpace(name), OwnerCount: 1, CreatedAt: now, UpdatedAt: now}
	if _, err = r.groups().InsertOne(ctx, g); err != nil {
		return nil, err
	}
	_, err = r.memberships().InsertOne(ctx, mongoMembership{GroupID: g.ID, UserID: userOID, Role: RoleOwner, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		_, _ = r.groups().DeleteOne(ctx, bson.M{"_id": g.ID})
		return nil, err
	}
	return &Group{ID: g.ID.Hex(), Name: g.Name, Role: RoleOwner, CreatedAt: now, UpdatedAt: now}, nil
}

func (r *RepositoryImpl) ListMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	groupOID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.memberships().Find(ctx, bson.M{"group_id": groupOID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []mongoMembership
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]GroupMember, 0, len(rows))
	for _, row := range rows {
		var u mongoModel
		if err = r.collection.FindOne(ctx, bson.M{"_id": row.UserID}).Decode(&u); err != nil {
			return nil, err
		}
		out = append(out, GroupMember{User: toDomainModel(&u), GroupID: groupID, Role: row.Role, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *RepositoryImpl) AddMember(ctx context.Context, groupID, email, password, role string) (*GroupMember, error) {
	if !validRole(role) || role == RoleOwner {
		return nil, errors.New("invalid role")
	}
	groupOID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, err
	}
	if err = r.groups().FindOne(ctx, bson.M{"_id": groupOID}).Err(); err != nil {
		return nil, err
	}
	existing, err := r.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	user := existing
	if user == nil {
		if password == "" {
			return nil, errors.New("a strong password is required for a new account")
		}
		user, err = r.Create(ctx, &Model{Email: email, Password: password, Active: true})
		if err != nil {
			return nil, err
		}
	}
	userOID, _ := primitive.ObjectIDFromHex(user.ID)
	now := time.Now().UTC()
	if _, err = r.memberships().InsertOne(ctx, mongoMembership{GroupID: groupOID, UserID: userOID, Role: role, CreatedAt: now, UpdatedAt: now}); err != nil {
		if existing == nil {
			_, _ = r.collection.DeleteOne(ctx, bson.M{"_id": userOID})
		}
		return nil, err
	}
	return &GroupMember{User: &Model{ID: user.ID, Email: user.Email, Active: true, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, GroupID: groupID, Role: role, CreatedAt: now}, nil
}

func (r *RepositoryImpl) UpdateMemberRole(ctx context.Context, groupID, userID, role string) error {
	if !validRole(role) {
		return errors.New("invalid role")
	}
	groupOID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return err
	}
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	var member mongoMembership
	if err = r.memberships().FindOne(ctx, bson.M{"group_id": groupOID, "user_id": userOID}).Decode(&member); err != nil {
		return err
	}
	if member.Role == RoleOwner && role != RoleOwner {
		res, updateErr := r.groups().UpdateOne(ctx,
			bson.M{"_id": groupOID, "owner_count": bson.M{"$gt": 1}},
			bson.M{"$inc": bson.M{"owner_count": int64(-1)}})
		if updateErr != nil {
			return updateErr
		}
		if res.ModifiedCount != 1 {
			return errors.New("cannot remove the last group owner")
		}
	}
	res, err := r.memberships().UpdateOne(ctx, bson.M{"group_id": groupOID, "user_id": userOID}, bson.M{"$set": bson.M{"role": role, "updatedAt": time.Now().UTC()}})
	if err == nil && res.MatchedCount == 0 {
		err = errors.New("group membership not found")
	}
	if err != nil && member.Role == RoleOwner && role != RoleOwner {
		_, _ = r.groups().UpdateOne(ctx, bson.M{"_id": groupOID}, bson.M{"$inc": bson.M{"owner_count": int64(1)}})
	}
	return err
}

func (r *RepositoryImpl) RemoveMember(ctx context.Context, groupID, userID string) error {
	groupOID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return err
	}
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	var member mongoMembership
	if err = r.memberships().FindOne(ctx, bson.M{"group_id": groupOID, "user_id": userOID}).Decode(&member); err != nil {
		return err
	}
	if member.Role == RoleOwner {
		res, updateErr := r.groups().UpdateOne(ctx,
			bson.M{"_id": groupOID, "owner_count": bson.M{"$gt": 1}},
			bson.M{"$inc": bson.M{"owner_count": int64(-1)}})
		if updateErr != nil {
			return updateErr
		}
		if res.ModifiedCount != 1 {
			return errors.New("cannot remove the last group owner")
		}
	}
	res, err := r.memberships().DeleteOne(ctx, bson.M{"group_id": groupOID, "user_id": userOID})
	if err == nil && res.DeletedCount == 0 {
		err = errors.New("group membership not found")
	}
	if err != nil && member.Role == RoleOwner {
		_, _ = r.groups().UpdateOne(ctx, bson.M{"_id": groupOID}, bson.M{"$inc": bson.M{"owner_count": int64(1)}})
	}
	return err
}

func (r *RepositoryImpl) ensureIdentityIndexes(ctx context.Context) {
	_, _ = r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	_, _ = r.memberships().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	_, _ = r.memberships().Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "user_id", Value: 1}}})
}
