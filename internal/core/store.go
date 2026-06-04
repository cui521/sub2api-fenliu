package core

import "context"

type AccountStore interface {
	UpsertAccount(ctx context.Context, account Account) (Account, error)
	ListAccounts(ctx context.Context) ([]Account, error)
	GetAccount(ctx context.Context, accountID string) (Account, bool, error)
	UpdateAccount(ctx context.Context, account Account) error
	ListCandidates(ctx context.Context, req SelectRequest) ([]Account, error)
	ListProbeDue(ctx context.Context) ([]Account, error)
}

type RuntimeStore interface {
	TryAcquireLease(ctx context.Context, accountID string, requestID string, maxConcurrency int, ttlSeconds int) (Lease, bool, error)
	ReleaseLease(ctx context.Context, leaseID string) error
	GetInflight(ctx context.Context, accountID string) int
	SetSticky(ctx context.Context, key string, accountID string, ttlSeconds int)
	GetSticky(ctx context.Context, key string) (string, bool)
	AcquireProbeLock(ctx context.Context, accountID string, ttlSeconds int) bool
	ReleaseProbeLock(ctx context.Context, accountID string)
}

type Lease struct {
	LeaseID   string
	AccountID string
	RequestID string
	ExpiresAt int64
}

type ProbeAdapter interface {
	Probe(ctx context.Context, account Account, model string) ProbeResult
}
