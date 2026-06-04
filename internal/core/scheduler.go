package core

import (
	"context"
	"errors"
	"sort"
	"time"
)

type Scheduler struct {
	store   AccountStore
	runtime RuntimeStore
	config  Config
}

func NewScheduler(store AccountStore, runtime RuntimeStore) *Scheduler {
	return &Scheduler{store: store, runtime: runtime, config: DefaultConfig()}
}

func (s *Scheduler) Select(ctx context.Context, req SelectRequest) (SelectResponse, error) {
	if req.RequestID == "" || req.Platform == "" {
		return SelectResponse{}, errors.New("request_id and platform are required")
	}

	candidates, err := s.store.ListCandidates(ctx, req)
	if err != nil {
		return SelectResponse{}, err
	}

	details := map[string]int{}
	dispatchable := make([]Account, 0, len(candidates))
	for _, account := range candidates {
		if !account.DispatchEnabled || (account.HealthStatus != HealthHealthy && account.HealthStatus != HealthSuspect) {
			details["disabled"]++
			continue
		}
		if s.runtime.GetInflight(ctx, account.AccountID) >= account.MaxConcurrency {
			details["concurrency_full"]++
			continue
		}
		dispatchable = append(dispatchable, account)
	}

	if len(dispatchable) == 0 {
		return SelectResponse{Decision: "no_account", Reason: "no_dispatchable_account", Details: details}, nil
	}

	stickyKey := stickyKey(req)
	if stickyAccountID, ok := s.runtime.GetSticky(ctx, stickyKey); ok {
		for _, account := range dispatchable {
			if account.AccountID == stickyAccountID {
				lease, acquired, err := s.runtime.TryAcquireLease(ctx, account.AccountID, req.RequestID, account.MaxConcurrency, int(s.config.LeaseTTL.Seconds()))
				if err != nil {
					return SelectResponse{}, err
				}
				if acquired {
					return SelectResponse{
						AccountID: account.AccountID, HealthStatus: account.HealthStatus, Decision: "selected",
						Reason: "sticky", LeaseID: lease.LeaseID, LeaseTTLSecond: int(s.config.LeaseTTL.Seconds()),
					}, nil
				}
			}
		}
		if req.RequireSticky {
			return SelectResponse{Decision: "no_account", Reason: "sticky_account_unavailable", Details: details}, nil
		}
	}

	sort.SliceStable(dispatchable, func(i, j int) bool {
		return s.score(ctx, dispatchable[i]) > s.score(ctx, dispatchable[j])
	})

	for _, account := range dispatchable {
		lease, acquired, err := s.runtime.TryAcquireLease(ctx, account.AccountID, req.RequestID, account.MaxConcurrency, int(s.config.LeaseTTL.Seconds()))
		if err != nil {
			return SelectResponse{}, err
		}
		if !acquired {
			continue
		}
		s.runtime.SetSticky(ctx, stickyKey, account.AccountID, int(s.config.StickyTTL.Seconds()))
		return SelectResponse{
			AccountID: account.AccountID, HealthStatus: account.HealthStatus, Decision: "selected",
			Reason: "highest_score", LeaseID: lease.LeaseID, LeaseTTLSecond: int(s.config.LeaseTTL.Seconds()),
		}, nil
	}

	return SelectResponse{Decision: "no_account", Reason: "lease_acquire_failed", Details: details}, nil
}

func (s *Scheduler) ReportSuccess(ctx context.Context, req ReportSuccessRequest) error {
	_ = s.runtime.ReleaseLease(ctx, req.LeaseID)
	account, ok, err := s.store.GetAccount(ctx, req.AccountID)
	if err != nil || !ok {
		return err
	}
	now := time.Now().UTC()
	account.ConsecutiveFailures = 0
	account.ProbeFailureCount = 0
	account.LastSuccessAt = &now
	if account.HealthStatus == HealthSuspect && account.SuspectUntil != nil && !account.SuspectUntil.After(now) {
		account.HealthStatus = HealthHealthy
		account.DispatchEnabled = true
	}
	return s.store.UpdateAccount(ctx, account)
}

func (s *Scheduler) ReportFailure(ctx context.Context, req ReportFailureRequest) error {
	_ = s.runtime.ReleaseLease(ctx, req.LeaseID)
	account, ok, err := s.store.GetAccount(ctx, req.AccountID)
	if err != nil || !ok {
		return err
	}

	errorType := normalizeErrorType(req.ErrorType)
	now := time.Now().UTC()
	account.ConsecutiveFailures++
	account.ConsecutiveSuccesses = 0
	account.LastErrorType = errorType
	account.LastErrorCode = req.ErrorCode
	account.LastErrorMessage = req.ErrorMessage
	account.LastFailedAt = &now

	if account.ConsecutiveFailures >= disableThreshold(errorType, s.config) {
		delay := probeDelay(errorType)
		nextProbeAt := now.Add(delay)
		disabledAt := now
		account.DispatchEnabled = false
		account.HealthStatus = HealthDisabled
		account.DisabledReason = errorType
		account.DisabledAt = &disabledAt
		account.NextProbeAt = &nextProbeAt
	}

	return s.store.UpdateAccount(ctx, account)
}

func (s *Scheduler) score(ctx context.Context, account Account) int {
	score := account.Priority*10 + account.Weight
	inflight := s.runtime.GetInflight(ctx, account.AccountID)
	if account.MaxConcurrency > 0 {
		score -= inflight * 500 / account.MaxConcurrency
	}
	if account.HealthStatus == HealthSuspect {
		score -= 300
	}
	score -= account.ConsecutiveFailures * 100
	return score
}

func stickyKey(req SelectRequest) string {
	if req.SessionKey == "" {
		return ""
	}
	return req.Platform + ":" + req.SessionKey
}

