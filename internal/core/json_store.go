package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type JSONAccountStore struct {
	mu       sync.RWMutex
	path     string
	accounts map[string]Account
}

func NewJSONAccountStore(path string) (*JSONAccountStore, error) {
	store := &JSONAccountStore{path: path, accounts: map[string]Account{}}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, store.flush()
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &store.accounts); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *JSONAccountStore) UpsertAccount(ctx context.Context, account Account) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if account.AccountID == "" {
		return Account{}, errors.New("account_id is required")
	}
	if account.HealthStatus == "" {
		account.HealthStatus = HealthHealthy
	}
	if account.MaxConcurrency <= 0 {
		account.MaxConcurrency = 1
	}
	if account.Priority == 0 {
		account.Priority = 100
	}
	if account.Weight == 0 {
		account.Weight = 100
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now
	if !account.DispatchEnabled && account.HealthStatus == HealthHealthy {
		account.HealthStatus = HealthDisabled
	}
	if account.HealthStatus == HealthHealthy || account.HealthStatus == HealthSuspect {
		account.DispatchEnabled = true
	}
	s.accounts[account.AccountID] = account
	return account, s.flushLocked()
}

func (s *JSONAccountStore) ListAccounts(ctx context.Context) ([]Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		out = append(out, account)
	}
	return out, nil
}

func (s *JSONAccountStore) GetAccount(ctx context.Context, accountID string) (Account, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	return account, ok, nil
}

func (s *JSONAccountStore) UpdateAccount(ctx context.Context, account Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account.UpdatedAt = time.Now().UTC()
	s.accounts[account.AccountID] = account
	return s.flushLocked()
}

func (s *JSONAccountStore) ListCandidates(ctx context.Context, req SelectRequest) ([]Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Account{}
	for _, account := range s.accounts {
		if account.Platform != req.Platform {
			continue
		}
		if req.GroupID != "" && account.GroupID != req.GroupID {
			continue
		}
		out = append(out, account)
	}
	return out, nil
}

func (s *JSONAccountStore) ListProbeDue(ctx context.Context) ([]Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	out := []Account{}
	for _, account := range s.accounts {
		if account.ProbePaused {
			continue
		}
		if account.NextProbeAt != nil && !account.NextProbeAt.After(now) {
			out = append(out, account)
		}
	}
	return out, nil
}

func (s *JSONAccountStore) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked()
}

func (s *JSONAccountStore) flushLocked() error {
	data, err := json.MarshalIndent(s.accounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
