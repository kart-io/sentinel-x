package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/internal/model"
)

// LogStore defines operations for login logs.
type LogStore struct {
	db *gorm.DB
}

// NewLogStore creates a new LogStore.
func NewLogStore(db *gorm.DB) *LogStore {
	return &LogStore{db: db}
}

// Create creates a new login log.
func (s *LogStore) Create(ctx context.Context, log *model.LoginLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}
