package monitor_group

import "context"

type Repository interface {
	Create(context.Context, *Model) (*Model, error)
	FindByID(context.Context, string) (*Model, error)
	FindByName(context.Context, string) (*Model, error)
	FindAll(context.Context) ([]*Model, error)
	Update(context.Context, string, *UpdateDto) (*Model, error)
	Delete(context.Context, string) error
}
