package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryRuntimeStore struct {
	mu         sync.Mutex
	leases     map[string]Lease
	sticky     map[string]stickyValue
	probeLocks map[string]time.Time
}

type stickyValue struct {
	AccountID string
	ExpiresAt time.Time
}

func NewMemoryRuntimeStore() *MemoryRuntimeStore {
	return &MemoryRuntimeStore{
		leases:     map[string]Lease{},
		sticky:     map[string]stickyValue{},
		probeLocks: map[string]time.Time{},
	}
}

func (s *MemoryRuntimeStore) TryAcquireLease(ctx context.Context, accountID string, requestID string, maxConcurrency int, ttlSeconds int) (Lease, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if s.inflightLocked(accountID) >= maxConcurrency {
		return Lease{}, false, nil
	}
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	lease := Lease{
		LeaseID:   fmt.Sprintf("lease_%s_%s", requestID, accountID),
		AccountID: accountID,
		RequestID: requestID,
		ExpiresAt: expiresAt.Unix(),
	}
	s.leases[lease.LeaseID] = lease
	return lease, true, nil
}

func (s *MemoryRuntimeStore) ReleaseLease(ctx context.Context, leaseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.leases, leaseID)
	return nil
}

func (s *MemoryRuntimeStore) GetInflight(ctx context.Context, accountID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	return s.inflightLocked(accountID)
}

func (s *MemoryRuntimeStore) SetSticky(ctx context.Context, key string, accountID string, ttlSeconds int) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sticky[key] = stickyValue{AccountID: accountID, ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second)}
}

func (s *MemoryRuntimeStore) GetSticky(ctx context.Context, key string) (string, bool) {
	if key == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.sticky[key]
	if !ok || time.Now().After(value.ExpiresAt) {
		delete(s.sticky, key)
		return "", false
	}
	return value.AccountID, true
}

func (s *MemoryRuntimeStore) AcquireProbeLock(ctx context.Context, accountID string, ttlSeconds int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if expiresAt, ok := s.probeLocks[accountID]; ok && expiresAt.After(now) {
		return false
	}
	s.probeLocks[accountID] = now.Add(time.Duration(ttlSeconds) * time.Second)
	return true
}

func (s *MemoryRuntimeStore) ReleaseProbeLock(ctx context.Context, accountID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.probeLocks, accountID)
}

func (s *MemoryRuntimeStore) inflightLocked(accountID string) int {
	count := 0
	for _, lease := range s.leases {
		if lease.AccountID == accountID {
			count++
		}
	}
	return count
}

func (s *MemoryRuntimeStore) cleanupLocked() {
	now := time.Now().Unix()
	for leaseID, lease := range s.leases {
		if lease.ExpiresAt <= now {
			delete(s.leases, leaseID)
		}
	}
	for key, value := range s.sticky {
		if time.Now().After(value.ExpiresAt) {
			delete(s.sticky, key)
		}
	}
}

