package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/scheduler/store"
	"github.com/kart-io/sentinel-x/pkg/utils/id"
)

// JobManager defines the business logic connection for jobs.
type JobManager struct {
	store store.JobStore
	cron  cron.Parser
}

// NewJobManager creates a new JobManager.
func NewJobManager(store store.JobStore) *JobManager {
	return &JobManager{
		store: store,
		cron:  cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// CreateJob creates a new job.
func (m *JobManager) CreateJob(ctx context.Context, job *model.Job) error {
	// Validate and parse cron expression
	schedule, err := m.cron.Parse(job.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	job.NextRunTime = schedule.Next(time.Now())
	
	// Generate ID if missing
	if job.ID == "" {
		job.ID = id.NewUUID() // Assuming this util exists based on project analysis
	}

	return m.store.Create(ctx, job)
}

// UpdateJob updates an existing job.
func (m *JobManager) UpdateJob(ctx context.Context, job *model.Job) error {
	// If cron expression is updated, re-calculate next run time
	if job.CronExpression != "" {
		schedule, err := m.cron.Parse(job.CronExpression)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		job.NextRunTime = schedule.Next(time.Now())
	}

	return m.store.Update(ctx, job)
}

// DeleteJob deletes a job.
func (m *JobManager) DeleteJob(ctx context.Context, id string) error {
	return m.store.Delete(ctx, id)
}

// GetJob retrieves a job.
func (m *JobManager) GetJob(ctx context.Context, id string) (*model.Job, error) {
	return m.store.Get(ctx, id)
}

// ListJobs lists jobs.
func (m *JobManager) ListJobs(ctx context.Context, offset, limit int) ([]*model.Job, int64, error) {
	return m.store.List(ctx, offset, limit)
}
