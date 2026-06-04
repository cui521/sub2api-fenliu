package core

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Sub2PostgresStore struct {
	pool *pgxpool.Pool
}

type sub2HealthExtra struct {
	HealthStatus         string     `json:"health_status,omitempty"`
	ConsecutiveFailures  int        `json:"consecutive_failures,omitempty"`
	ConsecutiveSuccesses int        `json:"consecutive_successes,omitempty"`
	ProbeFailureCount    int        `json:"probe_failure_count,omitempty"`
	ProbePaused          bool       `json:"probe_paused,omitempty"`
	ProbeTotalCount      int        `json:"probe_total_count,omitempty"`
	ProbeSuccessCount    int        `json:"probe_success_count,omitempty"`
	ProbeErrorCount      int        `json:"probe_error_count,omitempty"`
	LastProbeAt          *time.Time `json:"last_probe_at,omitempty"`
	LastErrorType        string     `json:"last_error_type,omitempty"`
	LastErrorCode        string     `json:"last_error_code,omitempty"`
	LastErrorMessage     string     `json:"last_error_message,omitempty"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty"`
	LastFailedAt         *time.Time `json:"last_failed_at,omitempty"`
	DisabledReason      string     `json:"disabled_reason,omitempty"`
	DisabledAt          *time.Time `json:"disabled_at,omitempty"`
	NextProbeAt         *time.Time `json:"next_probe_at,omitempty"`
	SuspectUntil         *time.Time `json:"suspect_until,omitempty"`
	ProbeURL            string     `json:"probe_url,omitempty"`
	ProbeMethod         string     `json:"probe_method,omitempty"`
	ProbeBody           string     `json:"probe_body,omitempty"`
}

type sub2Credentials struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

func NewSub2PostgresStore(ctx context.Context, dsn string) (*Sub2PostgresStore, error) {
	if dsn == "" {
		return nil, errors.New("postgres dsn is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Sub2PostgresStore{pool: pool}, nil
}

func (s *Sub2PostgresStore) UpsertAccount(ctx context.Context, account Account) (Account, error) {
	if account.AccountID == "" {
		return Account{}, errors.New("account_id is required")
	}
	existing, ok, err := s.GetAccount(ctx, account.AccountID)
	if err != nil {
		return Account{}, err
	}
	if !ok {
		return Account{}, errors.New("sub2 account not found; create it in sub2api first")
	}
	if account.ProbeURL != "" {
		existing.ProbeURL = account.ProbeURL
	}
	if account.ProbeMethod != "" {
		existing.ProbeMethod = account.ProbeMethod
	}
	if account.ProbeBody != "" {
		existing.ProbeBody = account.ProbeBody
	}
	if err := s.UpdateAccount(ctx, existing); err != nil {
		return Account{}, err
	}
	return existing, nil
}

func (s *Sub2PostgresStore) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, platform, status, schedulable, priority, concurrency,
		       error_message, created_at, updated_at, credentials, extra
		from accounts
		where deleted_at is null
		  and name not like '%@%'
		  and coalesce(credentials->>'api_key', '') <> ''
		order by id desc
		limit 2000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []Account{}
	for rows.Next() {
		account, err := scanSub2Account(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *Sub2PostgresStore) GetAccount(ctx context.Context, accountID string) (Account, bool, error) {
	id, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return Account{}, false, nil
	}
	row := s.pool.QueryRow(ctx, `
		select id, name, platform, status, schedulable, priority, concurrency,
		       error_message, created_at, updated_at, credentials, extra
		from accounts
		where id = $1
		  and deleted_at is null
		  and name not like '%@%'
		  and coalesce(credentials->>'api_key', '') <> ''`, id)
	account, err := scanSub2Account(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, false, nil
		}
		return Account{}, false, err
	}
	return account, true, nil
}

func (s *Sub2PostgresStore) UpdateAccount(ctx context.Context, account Account) error {
	health := sub2HealthExtra{
		HealthStatus: account.HealthStatus,
		ConsecutiveFailures: account.ConsecutiveFailures,
		ConsecutiveSuccesses: account.ConsecutiveSuccesses,
		ProbeFailureCount: account.ProbeFailureCount,
		ProbePaused: account.ProbePaused,
		ProbeTotalCount: account.ProbeTotalCount,
		ProbeSuccessCount: account.ProbeSuccessCount,
		ProbeErrorCount: account.ProbeErrorCount,
		LastProbeAt: account.LastProbeAt,
		LastErrorType: account.LastErrorType,
		LastErrorCode: account.LastErrorCode,
		LastErrorMessage: account.LastErrorMessage,
		LastSuccessAt: account.LastSuccessAt,
		LastFailedAt: account.LastFailedAt,
		DisabledReason: account.DisabledReason,
		DisabledAt: account.DisabledAt,
		NextProbeAt: account.NextProbeAt,
		SuspectUntil: account.SuspectUntil,
		ProbeURL: account.ProbeURL,
		ProbeMethod: account.ProbeMethod,
		ProbeBody: account.ProbeBody,
	}
	healthJSON, err := json.Marshal(health)
	if err != nil {
		return err
	}

	status := account.Status
	if status == "" {
		status = "active"
	}
	if !account.DispatchEnabled && account.DisabledReason != "" {
		status = "error"
	}

	_, err = s.pool.Exec(ctx, `
		update accounts
		set schedulable = $1,
		    status = $2,
		    error_message = $3,
		    temp_unschedulable_until = $4,
		    temp_unschedulable_reason = $5,
		    extra = jsonb_set(coalesce(extra, '{}'::jsonb), '{aad_health}', $6::jsonb, true),
		    updated_at = now()
		where id = $7 and deleted_at is null`,
		account.DispatchEnabled,
		status,
		nullableString(account.LastErrorMessage),
		account.NextProbeAt,
		nullableString(account.DisabledReason),
		string(healthJSON),
		account.AccountID,
	)
	return err
}

func (s *Sub2PostgresStore) ListCandidates(ctx context.Context, req SelectRequest) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, platform, status, schedulable, priority, concurrency,
		       error_message, created_at, updated_at, credentials, extra
		from accounts
		where deleted_at is null
		  and platform = $1
		  and name not like '%@%'
		  and coalesce(credentials->>'api_key', '') <> ''
		order by priority desc, id desc
		limit 2000`, req.Platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []Account{}
	for rows.Next() {
		account, err := scanSub2Account(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *Sub2PostgresStore) ListProbeDue(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, platform, status, schedulable, priority, concurrency,
		       error_message, created_at, updated_at, credentials, extra
		from accounts
		where deleted_at is null
		  and name not like '%@%'
		  and coalesce(credentials->>'api_key', '') <> ''
		  and coalesce((extra->'aad_health'->>'probe_paused')::boolean, false) = false
		  and (
		    (extra->'aad_health'->>'next_probe_at') is not null
		    and (extra->'aad_health'->>'next_probe_at')::timestamptz <= now()
		  )
		order by id asc
		limit 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []Account{}
	for rows.Next() {
		account, err := scanSub2Account(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

type sub2Scanner interface {
	Scan(dest ...any) error
}

func scanSub2Account(row sub2Scanner) (Account, error) {
	var id int64
	var name, platform, status string
	var schedulable bool
	var priority, concurrency int
	var errorMessage *string
	var createdAt, updatedAt time.Time
	var credentialsBytes, extraBytes []byte

	if err := row.Scan(&id, &name, &platform, &status, &schedulable, &priority, &concurrency, &errorMessage, &createdAt, &updatedAt, &credentialsBytes, &extraBytes); err != nil {
		return Account{}, err
	}

	account := Account{
		AccountID: strconv.FormatInt(id, 10),
		Name: name,
		Platform: platform,
		Status: status,
		DispatchEnabled: schedulable,
		Priority: priority,
		Weight: 100,
		MaxConcurrency: concurrency,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if account.MaxConcurrency <= 0 {
		account.MaxConcurrency = 1
	}
	if errorMessage != nil {
		account.LastErrorMessage = *errorMessage
	}

	health := parseSub2Health(extraBytes)
	applySub2Health(&account, health)
	credentials := parseSub2Credentials(credentialsBytes)
	account.CredentialAPIKey = credentials.APIKey
	account.CredentialBaseURL = credentials.BaseURL
	applyAutoProbe(&account)
	if account.HealthStatus == "" {
		if schedulable && status == "active" {
			account.HealthStatus = HealthHealthy
		} else {
			account.HealthStatus = HealthDisabled
		}
	}
	return account, nil
}

func parseSub2Credentials(credentialsBytes []byte) sub2Credentials {
	var credentials sub2Credentials
	_ = json.Unmarshal(credentialsBytes, &credentials)
	return credentials
}

func applyAutoProbe(account *Account) {
	if account.ProbeURL != "" {
		return
	}
	if account.Platform != "openai" || account.CredentialBaseURL == "" || account.CredentialAPIKey == "" {
		return
	}
	baseURL := strings.TrimRight(account.CredentialBaseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		account.ProbeURL = baseURL + "/models"
	} else {
		account.ProbeURL = baseURL + "/v1/models"
	}
	account.ProbeMethod = "GET"
	account.ProbeHeaders = map[string]string{
		"Authorization": "Bearer " + account.CredentialAPIKey,
		"Accept":        "application/json",
	}
}

func parseSub2Health(extraBytes []byte) sub2HealthExtra {
	var extra map[string]json.RawMessage
	if len(extraBytes) == 0 || json.Unmarshal(extraBytes, &extra) != nil {
		return sub2HealthExtra{}
	}
	var health sub2HealthExtra
	if raw, ok := extra["aad_health"]; ok {
		_ = json.Unmarshal(raw, &health)
	}
	return health
}

func applySub2Health(account *Account, health sub2HealthExtra) {
	account.HealthStatus = health.HealthStatus
	account.ConsecutiveFailures = health.ConsecutiveFailures
	account.ConsecutiveSuccesses = health.ConsecutiveSuccesses
	account.ProbeFailureCount = health.ProbeFailureCount
	account.ProbePaused = health.ProbePaused
	account.ProbeTotalCount = health.ProbeTotalCount
	account.ProbeSuccessCount = health.ProbeSuccessCount
	account.ProbeErrorCount = health.ProbeErrorCount
	account.LastProbeAt = health.LastProbeAt
	account.LastErrorType = health.LastErrorType
	account.LastErrorCode = health.LastErrorCode
	if health.LastErrorMessage != "" {
		account.LastErrorMessage = health.LastErrorMessage
	}
	account.LastSuccessAt = health.LastSuccessAt
	account.LastFailedAt = health.LastFailedAt
	account.DisabledReason = health.DisabledReason
	account.DisabledAt = health.DisabledAt
	account.NextProbeAt = health.NextProbeAt
	account.SuspectUntil = health.SuspectUntil
	account.ProbeURL = health.ProbeURL
	account.ProbeMethod = health.ProbeMethod
	account.ProbeBody = health.ProbeBody
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

