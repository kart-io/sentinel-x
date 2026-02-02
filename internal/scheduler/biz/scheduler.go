package biz

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/kart-io/logger"
	"github.com/robfig/cron/v3"
	"go.etcd.io/etcd/client/v3/concurrency"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/scheduler/store"
	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
)

const (
	electionKey   = "/sentinel/scheduler/leader"
	pollInterval  = 10 * time.Second
	prefetchDelta = 12 * time.Second // slightly larger than poll interval
)

// Scheduler responsible for polling and executing jobs.
type Scheduler struct {
	store      store.JobStore
	redis      *redis.Client
	etcd       *etcd.Client
	cronParser cron.Parser
	httpClient *http.Client
	stopCh     chan struct{}
	wg         sync.WaitGroup
	running    bool
	mu         sync.Mutex
	session    *concurrency.Session
	election   *concurrency.Election
}

// NewScheduler creates a new Scheduler.
func NewScheduler(store store.JobStore, redisClient *redis.Client, etcdClient *etcd.Client) *Scheduler {
	return &Scheduler{
		store:      store,
		redis:      redisClient,
		etcd:       etcdClient,
		cronParser: cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

// Start starts the scheduler polling loop.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	logger.Info("Scheduler started")
	s.wg.Add(1)
	go s.campaign(ctx)
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.running = false
	logger.Info("Scheduler stopped")
}

func (s *Scheduler) campaign(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Create a new session
		session, err := concurrency.NewSession(s.etcd.Raw(), concurrency.WithTTL(15))
		if err != nil {
			logger.Errorw("Failed to create etcd session", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		s.session = session
		s.election = concurrency.NewElection(session, electionKey)

		logger.Info("Campaigning for leadership")
		
		// Campaign blocks until leadership is acquired or error
		if err := s.election.Campaign(ctx, "scheduler-node"); err != nil {
			logger.Errorw("Campaign failed", "error", err)
			session.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		logger.Info("Acquired leadership, starting poll")
		
		// Run polling loop while leader
		s.leaderLoop(ctx)

		logger.Info("Lost leadership or resigning")
		// If we return from leaderLoop, verify session still valid then resign?
		// Usually leaderLoop returns when ctx canceled or stopCh closed or session expired.
		if s.election != nil {
			ctxResign, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			s.election.Resign(ctxResign)
			cancel()
		}
		if s.session != nil {
			s.session.Close()
		}
	}
}

func (s *Scheduler) leaderLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Initial fetch
	s.processDueJobsBuffered(ctx)

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-s.session.Done():
			logger.Warn("Etcd session expired")
			return
		case <-ticker.C:
			s.processDueJobsBuffered(ctx)
		}
	}
}

func (s *Scheduler) processDueJobsBuffered(ctx context.Context) {
	// Fetch jobs due between now and now + prefetchDelta
	// Note: We also pick up past due jobs that missed execution
	// But `ListDueJobs` typically gets `next_run_time <= now`.
	// We need to extend `ListDueJobs` or update specific query logic.
	// Since we are buffering, we want `next_run_time <= now + prefetchDelta`.
	// However, the current store method `ListDueJobs` likely does `next_run_time <= NOW()`.
	// We should probably rely on fetching `next_run_time <= time.Now().Add(prefetchDelta)`.
	// And we sort them in memory.
	
	// Since I cannot change store interface easily without breaking things or more lookups,
	// I will assume `ListDueJobs` implementation uses `time.Now()` internally or accepts a time limit.
	// Checking `store/job.go` would be ideal, but for now I will rely on standard behavior.
	// Wait, I designed it. `ListDueJobs` runs `next_run_time <= NOW()`.
	// So it won't fetch future jobs for buffering.
	// To implement buffering effectively without changing store, I can just poll every 1s (as before) OR
	// I must change store to accept a "until" time.
	
	// Let's stick to the plan of "Buffering", implying we fetch future jobs?
	// If I can't fetch future jobs (only past due), then buffering only helps batching PAST logic.
	// Ah, the goal was "Reduce Database Load".
	// If I poll every 10s, I will miss precision strictly speaking unless I fetch future jobs.
	// I WILL MODIFY THE QUERY logic directly here or assume I can update `ListDueJobs` to take a param?
	// I will check `store` implementation.
	
	// Actually, let's look at `store/job.go` later. For now, I'll pass a context? No.
	// Let's assume for this step I will execute what I get.
	// BUT, if I only run every 10s, jobs will be delayed by up to 10s.
	// To fix delay, we MUST look ahead.
	// I will update the `store` logic momentarily via `multi_replace`.
	// For now, I will write the code assuming `store.ListDueJobs` takes a generic logic OR I add `ListJobsDueBefore(ctx, timestamp)`.
	
	// Let's add `ListJobsDueBefore` to store interface in a separate step or just use `ListDueJobs` 
	// but modify `ListDueJobs` definition.
	// Let's assume `ListDueJobs` accepts a limit, but I need a time horizon.
	
	// Wait, I can't change interface without updating all files.
	// Let's write the Scheduler code to use `store.ListJobsDueBy(ctx, time.Now().Add(prefetchDelta), limit)`
	// and I will update store interface in next step.
	
	until := time.Now().Add(prefetchDelta)
	// We need to cast store to a type that has this method or update the interface.
	// I will update the interface.

	jobs, err := s.store.ListJobsDueBy(ctx, until, 100)
	if err != nil {
		logger.Errorw("Failed to list due jobs", "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	logger.Infow("Buffered jobs fetched", "count", len(jobs))

	for _, job := range jobs {
		// Calculate time to wait
		// If next_run_time is in past, wait = 0
		waitDuration := time.Until(job.NextRunTime)
		if waitDuration < 0 {
			waitDuration = 0
		}

		// Schedule execution
		s.wg.Add(1)
		time.AfterFunc(waitDuration, func() {
			defer s.wg.Done()
			s.executeJob(context.Background(), job)
		})
	}
}

func (s *Scheduler) executeJob(ctx context.Context, job *model.Job) {
	// Re-check if job is still valid or active? (Optimistic execution)
	// Since we buffered, job might have been deleted?
	// For MVP, we run it.
	
	logger.Infow("Executing job", "id", job.ID, "name", job.Name)

	schedule, err := s.cronParser.Parse(job.CronExpression)
	var nextRun time.Time
	if err == nil {
		nextRun = schedule.Next(time.Now())
	} else {
		logger.Errorw("Invalid cron during execution", "id", job.ID, "cron", job.CronExpression)
		nextRun = time.Now().Add(1 * time.Minute) 
	}

	if job.Target != "" {
		req, err := http.NewRequestWithContext(ctx, "POST", job.Target, bytes.NewBufferString(job.Payload))
		if err != nil {
			logger.Errorw("Failed to create request", "id", job.ID, "error", err)
		} else {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Job-ID", job.ID)
			
			resp, err := s.httpClient.Do(req)
			if err != nil {
				logger.Errorw("Job execution failed", "id", job.ID, "target", job.Target, "error", err)
			} else {
				resp.Body.Close()
				logger.Infow("Job executed", "id", job.ID, "status", resp.StatusCode)
			}
		}
	}

	now := time.Now()
	job.LastRunTime = &now
	job.NextRunTime = nextRun
	
	if err := s.store.Update(ctx, job); err != nil {
		logger.Errorw("Failed to update job after execution", "id", job.ID, "error", err)
	}
}
