package incident

import "context"

type Repository interface {
	Create(context.Context, string, string, string, string) (*Model, error)
	FindAll(context.Context) ([]*Model, error)
	AddUpdate(context.Context, string, string, string) (*Model, error)
	FindPublicByStatusPageSlug(context.Context, string) ([]*Model, error)
}
