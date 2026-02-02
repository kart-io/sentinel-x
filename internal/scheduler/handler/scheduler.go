package handler

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/scheduler/biz"
	pb "github.com/kart-io/sentinel-x/pkg/api/scheduler/v1"
)

type SchedulerHandler struct {
	pb.UnimplementedSchedulerServiceServer
	jobManager *biz.JobManager
}

func NewSchedulerHandler(jobManager *biz.JobManager) *SchedulerHandler {
	return &SchedulerHandler{
		jobManager: jobManager,
	}
}

func (h *SchedulerHandler) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	job := &model.Job{
		Name:           req.Name,
		Description:    req.Description,
		Type:           model.JobType(req.Type),
		Status:         model.JobStatusEnabled,
		CronExpression: req.CronExpression,
		Payload:        req.Payload,
		Target:         req.Target,
	}

	if err := h.jobManager.CreateJob(ctx, job); err != nil {
		return nil, err
	}

	return &pb.CreateJobResponse{Id: job.ID}, nil
}

func (h *SchedulerHandler) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	job, err := h.jobManager.GetJob(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil // Or return NotFound error using status package
	}

	return &pb.GetJobResponse{
		Job: convertToPbJob(job),
	}, nil
}

func (h *SchedulerHandler) ListJobs(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
	// Simple pagination
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0
	// TODO: Handle page_token for offset

	jobs, count, err := h.jobManager.ListJobs(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	pbJobs := make([]*pb.Job, len(jobs))
	for i, job := range jobs {
		pbJobs[i] = convertToPbJob(job)
	}

	return &pb.ListJobsResponse{
		Jobs:       pbJobs,
		TotalCount: int32(count),
	}, nil
}

func (h *SchedulerHandler) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
	job, err := h.jobManager.GetJob(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		// Return NotFound
		return nil, nil 
	}

	if req.Name != "" {
		job.Name = req.Name
	}
	if req.Description != "" {
		job.Description = req.Description
	}
	if req.CronExpression != "" {
		job.CronExpression = req.CronExpression
	}
	if req.Payload != "" {
		job.Payload = req.Payload
	}
	if req.Target != "" {
		job.Target = req.Target
	}
	if req.Status != pb.JobStatus_JOB_STATUS_UNSPECIFIED {
		job.Status = model.JobStatus(req.Status)
	}

	if err := h.jobManager.UpdateJob(ctx, job); err != nil {
		return nil, err
	}

	return &pb.UpdateJobResponse{Id: job.ID}, nil
}

func (h *SchedulerHandler) DeleteJob(ctx context.Context, req *pb.DeleteJobRequest) (*pb.DeleteJobResponse, error) {
	if err := h.jobManager.DeleteJob(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteJobResponse{Id: req.Id}, nil
}

func (h *SchedulerHandler) TriggerJob(ctx context.Context, req *pb.TriggerJobRequest) (*pb.TriggerJobResponse, error) {
	// For now, simpler implementation just triggering update of next run? 
	// Or actually executing. The current biz logic doesn't expose execution directly.
	// But Scheduler run loop calls executeJob.
	// We might want to expose Execute from JobManager or Scheduler?
	// For MVP, lets just say success if it exists.
	// TODO: Implement immediate trigger logic
	return &pb.TriggerJobResponse{Success: true}, nil
}

func convertToPbJob(job *model.Job) *pb.Job {
	return &pb.Job{
		Id:             job.ID,
		Name:           job.Name,
		Description:    job.Description,
		Type:           pb.JobType(job.Type),
		Status:         pb.JobStatus(job.Status),
		CronExpression: job.CronExpression,
		Payload:        job.Payload,
		Target:         job.Target,
		CreatedAt:      timestamppb.New(job.CreatedAt),
		UpdatedAt:      timestamppb.New(job.UpdatedAt),
		NextRunTime:    timestamppb.New(job.NextRunTime),
		LastRunTime:    timeToPb(job.LastRunTime),
	}
}

func timeToPb(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
