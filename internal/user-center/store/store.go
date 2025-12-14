package store

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/store"
)

// Factory defines the factory interface for creating stores.
type Factory interface {
	Users() UserStore
	Roles() RoleStore
	Close() error
}

// UserStore defines the user storage interface.
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, username string) error
	Get(ctx context.Context, username string) (*model.User, error)
	GetByUserID(ctx context.Context, userID uint64) (*model.User, error)
	List(ctx context.Context, opts ...store.Option) (int64, []*model.User, error)
}

// RoleStore defines the role storage interface.
type RoleStore interface {
	Create(ctx context.Context, role *model.Role) error
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, code string) error
	Get(ctx context.Context, code string) (*model.Role, error)
	List(ctx context.Context, opts ...store.Option) (int64, []*model.Role, error)
	AssignRole(ctx context.Context, userID, roleID uint64) error
	RevokeRole(ctx context.Context, userID, roleID uint64) error
	ListByUserID(ctx context.Context, userID uint64) ([]*model.Role, error)
}
