package auth

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, user *Model) (*Model, error)
	CreateFirstOwner(ctx context.Context, user *Model, groupName string) (*Model, *Group, error)
	FindByEmail(ctx context.Context, email string) (*Model, error)
	FindByID(ctx context.Context, id string) (*Model, error)
	FindAllCount(ctx context.Context) (int64, error)
	Update(ctx context.Context, id string, entity *UpdateModel) error
	ResolveMembership(ctx context.Context, userID, groupID string) (*Membership, error)
	ListGroups(ctx context.Context, userID string) ([]Group, error)
	CreateGroup(ctx context.Context, userID, name string) (*Group, error)
	ListMembers(ctx context.Context, groupID string) ([]GroupMember, error)
	AddMember(ctx context.Context, groupID, email, password, role string) (*GroupMember, error)
	UpdateMemberRole(ctx context.Context, groupID, userID, role string) error
	RemoveMember(ctx context.Context, groupID, userID string) error
}
