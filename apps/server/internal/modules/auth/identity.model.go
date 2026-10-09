package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	LegacyGroupID = "00000000-0000-4000-8000-000000000001"
	RoleOwner     = "owner"
	RoleAdmin     = "admin"
	RoleMember    = "member"
	RoleViewer    = "viewer"
)

type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Membership struct {
	GroupID string `json:"groupId"`
	UserID  string `json:"userId"`
	Role    string `json:"role"`
}

type GroupMember struct {
	User      *Model    `json:"user"`
	GroupID   string    `json:"groupId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type AddMemberDto struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role" validate:"required,oneof=admin member viewer"`
}

type UpdateMemberRoleDto struct {
	Role string `json:"role" validate:"required,oneof=admin member viewer"`
}

type CreateGroupDto struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type identityContextKey string

const (
	groupIDContextKey identityContextKey = "group_id"
	roleContextKey    identityContextKey = "group_role"
)

// WithIdentity attaches the authenticated workspace and role to repository
// contexts so feature modules can apply workspace filters without depending on
// Gin.
func WithIdentity(ctx context.Context, groupID, role string) context.Context {
	ctx = context.WithValue(ctx, groupIDContextKey, groupID)
	return context.WithValue(ctx, roleContextKey, role)
}

func GroupIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(groupIDContextKey).(string)
	return value, ok && value != ""
}

func RoleFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(roleContextKey).(string)
	return value, ok && value != ""
}

func MongoGroupIDFromContext(ctx context.Context) (primitive.ObjectID, bool, error) {
	groupID, ok := GroupIDFromContext(ctx)
	if !ok {
		return primitive.NilObjectID, false, nil
	}
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return primitive.NilObjectID, true, err
	}
	return objectID, true, nil
}

func validRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}
