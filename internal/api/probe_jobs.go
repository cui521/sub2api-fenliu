package api

import (
	"fmt"
	"sync"
	"time"

	"account-auto-dispatch/internal/core"
)

const (
	probeJobRunning   = "running"
	probeJobSucceeded = "succeeded"
	probeJobFailed    = "failed"
)

type probeJob struct {
	ID              string     `json:"job_id"`
	Status          string     `json:"status"`
	AccountID       string     `json:"account_id"`
	Name            string     `json:"name,omitempty"`
	Platform        string     `json:"platform,omitempty"`
	Model           string     `json:"model,omitempty"`
	Source          string     `json:"source"`
	Trigger         string     `json:"trigger,omitempty"`
	DryRun          bool       `json:"dry_run"`
	Success         bool       `json:"success"`
	ErrorType       string     `json:"error_type,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	Error           string     `json:"error,omitempty"`
	Request         any        `json:"request,omitempty"`
	Response        any        `json:"response,omitempty"`
	Events          any        `json:"events,omitempty"`
	DispatchEnabled bool       `json:"dispatch_enabled"`
	HealthStatus    string     `json:"health_status,omitempty"`
	LastSuccessAt    *time.Time `json:"last_success_at,omitempty"`
	NextProbeAt      *time.Time `json:"next_probe_at,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	ElapsedMS        int64      `json:"elapsed_ms"`
}

type probeJobStore struct {
	mu    sync.Mutex
	seq   uint64
	jobs  map[string]*probeJob
	order []string
}

func newProbeJobStore() *probeJobStore {
	return &probeJobStore{jobs: make(map[string]*probeJob)}
}

func (s *probeJobStore) Start(accountID string, model string, trigger string) probeJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seq++
	id := fmt.Sprintf("%d-%d", now.UnixNano(), s.seq)
	job := &probeJob{
		ID:        id,
		Status:    probeJobRunning,
		AccountID: accountID,
		Model:     model,
		Source:    "sub2_account_test",
		Trigger:   trigger,
		DryRun:    true,
		StartedAt: now,
		UpdatedAt: now,
	}
	s.jobs[id] = job
	s.order = append([]string{id}, s.order...)
	s.trimLocked(100)
	return cloneProbeJob(job)
}

func (s *probeJobStore) IsAccountRunning(accountID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.AccountID == accountID && job.Status == probeJobRunning {
			return true
		}
	}
	return false
}

func (s *probeJobStore) Finish(jobID string, account core.Account, result core.ProbeResult, accountFound bool, probeErr error) probeJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	result = core.RedactProbeResult(result)
	now := time.Now().UTC()
	job, ok := s.jobs[jobID]
	if !ok {
		job = &probeJob{
			ID:        jobID,
			Status:    probeJobRunning,
			AccountID: account.AccountID,
			Source:    "sub2_account_test",
			DryRun:    true,
			StartedAt: now,
		}
		s.jobs[jobID] = job
		s.order = append([]string{jobID}, s.order...)
	}

	if accountFound {
		job.AccountID = account.AccountID
		job.Name = account.Name
		job.Platform = account.Platform
		job.DispatchEnabled = account.DispatchEnabled
		job.HealthStatus = account.HealthStatus
		job.LastSuccessAt = account.LastSuccessAt
		job.NextProbeAt = account.NextProbeAt
	}

	job.Success = result.Success
	job.ErrorType = result.ErrorType
	job.ErrorCode = result.ErrorCode
	job.ErrorMessage = result.ErrorMessage
	job.Request = result.Request
	job.Response = result.Response
	job.Events = result.Events
	job.UpdatedAt = now
	job.FinishedAt = &now
	job.ElapsedMS = now.Sub(job.StartedAt).Milliseconds()

	if probeErr != nil {
		job.Status = probeJobFailed
		job.Error = core.RedactSensitiveText(probeErr.Error())
		if job.ErrorMessage == "" {
			job.ErrorMessage = core.RedactSensitiveText(probeErr.Error())
		}
	} else if result.Success {
		job.Status = probeJobSucceeded
	} else {
		job.Status = probeJobFailed
	}

	s.trimLocked(100)
	return cloneProbeJob(job)
}

func (s *probeJobStore) Get(jobID string) (probeJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return probeJob{}, false
	}
	return cloneProbeJobWithElapsed(job, time.Now().UTC()), true
}

func (s *probeJobStore) List(limit int) []probeJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.order) {
		limit = len(s.order)
	}
	now := time.Now().UTC()
	jobs := make([]probeJob, 0, limit)
	for _, id := range s.order[:limit] {
		if job, ok := s.jobs[id]; ok {
			jobs = append(jobs, cloneProbeJobWithElapsed(job, now))
		}
	}
	return jobs
}

func (s *probeJobStore) trimLocked(max int) {
	if len(s.order) <= max {
		return
	}
	for _, id := range s.order[max:] {
		delete(s.jobs, id)
	}
	s.order = s.order[:max]
}

func cloneProbeJob(job *probeJob) probeJob {
	return cloneProbeJobWithElapsed(job, time.Now().UTC())
}

func cloneProbeJobWithElapsed(job *probeJob, now time.Time) probeJob {
	copy := *job
	if copy.Status == probeJobRunning {
		copy.ElapsedMS = now.Sub(copy.StartedAt).Milliseconds()
	}
	return copy
}
