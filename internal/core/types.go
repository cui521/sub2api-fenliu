package core

import "time"

const (
	HealthHealthy  = "healthy"
	HealthSuspect  = "suspect"
	HealthDisabled = "disabled"
	HealthProbing  = "probing"
)

const (
	ErrAuthError      = "auth_error"
	ErrRateLimited    = "rate_limited"
	ErrQuotaExhausted = "quota_exhausted"
	ErrOverloaded     = "overloaded"
	ErrTimeout        = "timeout"
	ErrNetworkError   = "network_error"
	ErrServerError    = "server_error"
	ErrModelError     = "model_error"
	ErrUnknown        = "unknown_error"
)

type Config struct {
	CheckerScanInterval      time.Duration
	HealthyProbeInterval     time.Duration
	SuspectProbeInterval     time.Duration
	MinDisabledDuration      time.Duration
	RecoverySuccessThreshold int
	DisableFailureThreshold  int
	SuspectWindow            time.Duration
	MaxBackoff               time.Duration
	LeaseTTL                 time.Duration
	StickyTTL                time.Duration
	ProbeLimitPerScan        int
}

func DefaultConfig() Config {
	return Config{
		CheckerScanInterval:      30 * time.Second,
		HealthyProbeInterval:     10 * time.Minute,
		SuspectProbeInterval:     2 * time.Minute,
		MinDisabledDuration:      60 * time.Second,
		RecoverySuccessThreshold: 2,
		DisableFailureThreshold:  3,
		SuspectWindow:            5 * time.Minute,
		MaxBackoff:               60 * time.Minute,
		LeaseTTL:                 10 * time.Minute,
		StickyTTL:                24 * time.Hour,
		ProbeLimitPerScan:        1,
	}
}

type Account struct {
	AccountID            string            `json:"account_id"`
	Name                 string            `json:"name,omitempty"`
	Platform             string            `json:"platform"`
	GroupID              string            `json:"group_id,omitempty"`
	Status               string            `json:"status,omitempty"`
	HealthStatus         string            `json:"health_status"`
	DispatchEnabled      bool              `json:"dispatch_enabled"`
	Priority             int               `json:"priority"`
	Weight               int               `json:"weight"`
	MaxConcurrency       int               `json:"max_concurrency"`
	ConsecutiveFailures  int               `json:"consecutive_failures"`
	ConsecutiveSuccesses int               `json:"consecutive_successes"`
	ProbeFailureCount    int               `json:"probe_failure_count"`
	ProbePaused          bool              `json:"probe_paused,omitempty"`
	ProbeTotalCount      int               `json:"probe_total_count,omitempty"`
	ProbeSuccessCount    int               `json:"probe_success_count,omitempty"`
	ProbeErrorCount      int               `json:"probe_error_count,omitempty"`
	LastProbeAt          *time.Time        `json:"last_probe_at,omitempty"`
	LastErrorType        string            `json:"last_error_type,omitempty"`
	LastErrorCode        string            `json:"last_error_code,omitempty"`
	LastErrorMessage     string            `json:"last_error_message,omitempty"`
	LastSuccessAt        *time.Time        `json:"last_success_at,omitempty"`
	LastFailedAt         *time.Time        `json:"last_failed_at,omitempty"`
	DisabledReason      string            `json:"disabled_reason,omitempty"`
	DisabledAt          *time.Time        `json:"disabled_at,omitempty"`
	NextProbeAt         *time.Time        `json:"next_probe_at,omitempty"`
	SuspectUntil         *time.Time        `json:"suspect_until,omitempty"`
	ProbeURL            string            `json:"probe_url,omitempty"`
	ProbeMethod         string            `json:"probe_method,omitempty"`
	ProbeHeaders        map[string]string `json:"-"`
	ProbeBody           string            `json:"probe_body,omitempty"`
	CredentialAPIKey     string            `json:"-"`
	CredentialBaseURL    string            `json:"-"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type SelectRequest struct {
	RequestID             string `json:"request_id"`
	Platform              string `json:"platform"`
	GroupID               string `json:"group_id,omitempty"`
	Model                 string `json:"model,omitempty"`
	SessionKey            string `json:"session_key,omitempty"`
	RequireSticky         bool   `json:"require_sticky,omitempty"`
	EstimatedInputTokens  int    `json:"estimated_input_tokens,omitempty"`
	EstimatedOutputTokens int    `json:"estimated_output_tokens,omitempty"`
}

type SelectResponse struct {
	AccountID       string         `json:"account_id,omitempty"`
	HealthStatus   string         `json:"health_status,omitempty"`
	Decision       string         `json:"decision"`
	Reason         string         `json:"reason"`
	LeaseID        string         `json:"lease_id,omitempty"`
	LeaseTTLSecond int            `json:"lease_ttl_seconds,omitempty"`
	Details        map[string]int `json:"details,omitempty"`
}

type ReportSuccessRequest struct {
	RequestID    string `json:"request_id"`
	LeaseID      string `json:"lease_id"`
	AccountID    string `json:"account_id"`
	Platform     string `json:"platform"`
	Model        string `json:"model,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	LatencyMS    int    `json:"latency_ms,omitempty"`
}

type ReportFailureRequest struct {
	RequestID     string `json:"request_id"`
	LeaseID       string `json:"lease_id"`
	AccountID     string `json:"account_id"`
	Platform      string `json:"platform"`
	Model         string `json:"model,omitempty"`
	ErrorType     string `json:"error_type"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type ProbeRequest struct {
	AccountID string `json:"account_id"`
	Platform  string `json:"platform,omitempty"`
	Model     string `json:"model,omitempty"`
}

type ProbeResult struct {
	Success      bool              `json:"success"`
	ErrorType    string            `json:"error_type,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Request      *ProbeHTTPTrace   `json:"request,omitempty"`
	Response     *ProbeHTTPTrace   `json:"response,omitempty"`
	Events       []ProbeEventTrace `json:"events,omitempty"`
}

type ProbeHTTPTrace struct {
	Method     string            `json:"method,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	Status     string            `json:"status,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	BodySample string            `json:"body_sample,omitempty"`
}

type ProbeEventTrace struct {
	Event string `json:"event,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  string `json:"data,omitempty"`
}
