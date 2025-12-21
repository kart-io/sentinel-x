package store

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
)

// Factory defines the factory interface for creating stores.
type Factory interface {
	Users() UserStore
	AutoMigrate() error
	Close() error
}

// UserStore defines the user storage interface.
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, username string) error
	Get(ctx context.Context, username string) (*model.User, error)
	GetByUserID(ctx context.Context, userID uint64) (*model.User, error)
}
