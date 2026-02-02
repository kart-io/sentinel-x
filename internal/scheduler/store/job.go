package store

import (
	"context"
	"time"

	"github.com/kart-io/sentinel-x/internal/model"
	"gorm.io/gorm"
)

// JobStore defines the connection to the jobs storage.
type JobStore interface {
	Create(ctx context.Context, job *model.Job) error
	Get(ctx context.Context, id string) (*model.Job, error)
	List(ctx context.Context, offset, limit int) ([]*model.Job, int64, error)
	Update(ctx context.Context, job *model.Job) error
	Delete(ctx context.Context, id string) error
	ListDueJobs(ctx context.Context, limit int) ([]*model.Job, error)
	ListJobsDueBy(ctx context.Context, until time.Time, limit int) ([]*model.Job, error)
}

type jobStore struct {
	db *gorm.DB
}

// NewJobStore creates a new JobStore.
func NewJobStore(db *gorm.DB) JobStore {
	return &jobStore{db: db}
}

func (s *jobStore) Create(ctx context.Context, job *model.Job) error {
	return s.db.WithContext(ctx).Create(job).Error
}

func (s *jobStore) Get(ctx context.Context, id string) (*model.Job, error) {
	var job model.Job
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *jobStore) List(ctx context.Context, offset, limit int) ([]*model.Job, int64, error) {
	var jobs []*model.Job
	var count int64
	db := s.db.WithContext(ctx).Model(&model.Job{})
	
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&jobs).Error; err != nil {
		return nil, 0, err
	}
	
	return jobs, count, nil
}

func (s *jobStore) Update(ctx context.Context, job *model.Job) error {
	return s.db.WithContext(ctx).Save(job).Error
}

func (s *jobStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&model.Job{}, "id = ?", id).Error
}

func (s *jobStore) ListDueJobs(ctx context.Context, limit int) ([]*model.Job, error) {
	return s.ListJobsDueBy(ctx, time.Now(), limit)
}

func (s *jobStore) ListJobsDueBy(ctx context.Context, until time.Time, limit int) ([]*model.Job, error) {
	var jobs []*model.Job
	// status = 1 (Enabled)
	// next_run_time <= until
	// deleted_at IS NULL (handled by GORM usually if Model used, but explicit is good)
	if err := s.db.WithContext(ctx).Where("status = 1 AND next_run_time <= ?", until).Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}
