package core

import (
	"context"
	"log"
	"math"
	"time"
)

type HealthChecker struct {
	store  AccountStore
	prober ProbeAdapter
	config Config
}

func NewHealthChecker(store AccountStore, prober ProbeAdapter, config Config) *HealthChecker {
	return &HealthChecker{store: store, prober: prober, config: config}
}

func (c *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(c.config.CheckerScanInterval)
	defer ticker.Stop()
	for {
		c.ScanOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *HealthChecker) ScanOnce(ctx context.Context) {
	accounts, err := c.store.ListProbeDue(ctx)
	if err != nil {
		log.Printf("health checker list failed: %v", err)
		return
	}
	limit := c.config.ProbeLimitPerScan
	if limit <= 0 {
		limit = 1
	}
	for _, account := range accounts {
		c.ProbeAccount(ctx, account.AccountID, "")
		limit--
		if limit <= 0 {
			return
		}
	}
}

func (c *HealthChecker) ProbeAccount(ctx context.Context, accountID string, model string) ProbeResult {
	account, ok, err := c.store.GetAccount(ctx, accountID)
	if err != nil || !ok {
		return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: "account not found"}
	}

	result := c.prober.Probe(ctx, account, model)
	now := time.Now().UTC()

	if result.Success {
		account.ConsecutiveSuccesses++
		account.ConsecutiveFailures = 0
		account.ProbeFailureCount = 0
		account.LastErrorType = ""
		account.LastErrorCode = ""
		account.LastErrorMessage = ""
		account.LastSuccessAt = &now
		if account.ConsecutiveSuccesses >= c.config.RecoverySuccessThreshold {
			suspectUntil := now.Add(c.config.SuspectWindow)
			nextProbeAt := now.Add(c.config.SuspectProbeInterval)
			account.DispatchEnabled = true
			account.HealthStatus = HealthSuspect
			account.DisabledReason = ""
			account.DisabledAt = nil
			account.SuspectUntil = &suspectUntil
			account.NextProbeAt = &nextProbeAt
		} else {
			nextProbeAt := now.Add(probeDelay(account.LastErrorType))
			account.NextProbeAt = &nextProbeAt
		}
		_ = c.store.UpdateAccount(ctx, account)
		return result
	}

	errorType := normalizeErrorType(result.ErrorType)
	account.ConsecutiveFailures++
	account.ConsecutiveSuccesses = 0
	account.ProbeFailureCount++
	account.DispatchEnabled = false
	account.HealthStatus = HealthDisabled
	account.LastErrorType = errorType
	account.LastErrorCode = result.ErrorCode
	account.LastErrorMessage = humanErrorMessage(errorType, result.ErrorMessage)
	account.LastFailedAt = &now
	account.DisabledReason = errorType
	if account.DisabledAt == nil {
		account.DisabledAt = &now
	}
	nextProbeAt := now.Add(backoff(errorType, account.ProbeFailureCount, c.config.MaxBackoff))
	account.NextProbeAt = &nextProbeAt
	_ = c.store.UpdateAccount(ctx, account)
	return result
}

func (c *HealthChecker) ProbeAccountDryRun(ctx context.Context, accountID string, model string) (Account, ProbeResult, bool, error) {
	account, ok, err := c.store.GetAccount(ctx, accountID)
	if err != nil || !ok {
		return Account{}, ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: "account not found"}, ok, err
	}
	return account, c.prober.Probe(ctx, account, model), true, nil
}

func humanErrorMessage(errorType string, fallback string) string {
	switch errorType {
	case ErrAuthError:
		return "identity authentication failed: 401/403 unauthorized"
	case ErrRateLimited:
		return "rate limited by upstream"
	case ErrQuotaExhausted:
		return "quota exhausted"
	case ErrOverloaded:
		return "upstream overloaded"
	case ErrTimeout:
		return "probe request timeout"
	case ErrNetworkError:
		return "probe network error"
	case ErrServerError:
		return "upstream server error"
	case ErrModelError:
		return "model unavailable"
	default:
		if fallback != "" {
			return fallback
		}
		return "unknown probe error"
	}
}

func normalizeErrorType(errorType string) string {
	switch errorType {
	case ErrAuthError, ErrRateLimited, ErrQuotaExhausted, ErrOverloaded, ErrTimeout, ErrNetworkError, ErrServerError, ErrModelError:
		return errorType
	default:
		return ErrUnknown
	}
}

func disableThreshold(errorType string, config Config) int {
	switch errorType {
	case ErrAuthError, ErrQuotaExhausted, ErrModelError:
		return 1
	case ErrRateLimited, ErrOverloaded:
		return 2
	default:
		return config.DisableFailureThreshold
	}
}

func probeDelay(errorType string) time.Duration {
	switch errorType {
	case ErrAuthError:
		return 10 * time.Minute
	case ErrQuotaExhausted:
		return 30 * time.Minute
	case ErrRateLimited:
		return 5 * time.Minute
	case ErrOverloaded, ErrTimeout, ErrNetworkError:
		return time.Minute
	case ErrServerError, ErrUnknown:
		return 2 * time.Minute
	case ErrModelError:
		return 10 * time.Minute
	default:
		return 2 * time.Minute
	}
}

func backoff(errorType string, failureCount int, max time.Duration) time.Duration {
	if failureCount < 1 {
		failureCount = 1
	}
	base := probeDelay(errorType)
	multiplier := math.Pow(2, float64(failureCount-1))
	delay := time.Duration(float64(base) * multiplier)
	if delay > max {
		return max
	}
	return delay
}
