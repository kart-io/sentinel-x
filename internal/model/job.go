package model

import (
	"time"

	"gorm.io/gorm"
)

// JobType defines the type of the job.
type JobType int

const (
	JobTypeUnspecified JobType = 0
	JobTypeHTTP        JobType = 1
	JobTypeGRPC        JobType = 2
)

// JobStatus represents the current state of a job.
type JobStatus int

const (
	JobStatusUnspecified JobStatus = 0
	JobStatusEnabled     JobStatus = 1
	JobStatusDisabled    JobStatus = 2
)

// Job represents a scheduled task.
type Job struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)"`
	Name           string    `gorm:"type:varchar(255);not null"`
	Description    string    `gorm:"type:text"`
	Type           JobType   `gorm:"type:smallint;not null;default:1"`
	Status         JobStatus `gorm:"type:smallint;not null;default:1"`
	CronExpression string    `gorm:"type:varchar(255);not null"`
	Payload        string    `gorm:"type:text"`
	Target         string    `gorm:"type:varchar(1024);not null"`
	NextRunTime    time.Time `gorm:"index"`
	LastRunTime    *time.Time
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName returns the table name for the Job model.
func (Job) TableName() string {
	return "jobs"
}
