# Account Auto Dispatch Module

This document defines a standalone account auto-dispatch module that can be
embedded into an existing API gateway or deployed as a sidecar service.

The module owns account health, automatic scheduling enable/disable, probing,
and account selection. The host system only needs to call the module before and
after each upstream request.

## Goals

- Automatically enable dispatch for usable accounts.
- Automatically disable dispatch after consecutive failures.
- Periodically probe disabled accounts and re-enable them when recovered.
- Avoid dispatch flapping with cooldown, backoff, and success thresholds.
- Keep the integration surface small and platform-agnostic.

## Non Goals

- The module does not store upstream secrets unless used as the account store.
- The module does not proxy model traffic.
- The module does not replace billing, API key authentication, or user quota
  logic in the host system.

## Integration Model

The host system integrates at three points:

1. Select an account before sending an upstream request.
2. Report request success or failure after receiving the upstream result.
3. Let the module run a background health checker.

```text
client request
  -> host gateway
  -> account-auto-dispatch.select
  -> host sends request with selected account
  -> account-auto-dispatch.report_success / report_failure
  -> host returns response
```

## Account Health States

```text
healthy   Account is dispatchable.
suspect   Account is dispatchable but has reduced weight.
disabled  Account is not dispatchable.
probing   Account is being probed and is not dispatchable.
```

Dispatch rule:

```text
dispatch_enabled = health_status in (healthy, suspect)
```

## Recommended Defaults

```text
checker_scan_interval        30 seconds
healthy_probe_interval       10 minutes
suspect_probe_interval       2 minutes
min_disabled_duration        60 seconds
recovery_success_threshold   2 consecutive probe successes
disable_failure_threshold    3 consecutive ordinary failures
suspect_window               5 minutes
max_backoff                  60 minutes
```

For large account pools, set `healthy_probe_interval` to `0` to disable active
probing for healthy accounts. Healthy accounts can then rely on real traffic
feedback only.

## Failure Classification

The host should normalize upstream errors before reporting them.

```text
auth_error        401, 403, invalid token, account banned
rate_limited      429, too many requests
quota_exhausted   insufficient quota, balance exhausted, billing issue
overloaded        529, upstream overloaded
timeout           request timeout
network_error     DNS, connection reset, TLS failure
server_error      5xx except overloaded
model_error       model unavailable for this account
unknown_error     fallback type
```

## Disable Rules

```text
auth_error        disable after 1 failure
quota_exhausted   disable after 1 failure
rate_limited      disable after 2 consecutive failures
overloaded        disable after 2 consecutive failures
timeout           disable after 3 consecutive failures
network_error     disable after 3 consecutive failures
server_error      disable after 3 consecutive failures
model_error       disable after 1 failure for that model, optional account disable
unknown_error     disable after 3 consecutive failures
```

## Probe Delay Rules

```text
auth_error        10 minutes
quota_exhausted   30 minutes, or next billing reset if known
rate_limited      5 minutes
overloaded        1 minute
timeout           1 minute
network_error     1 minute
server_error      2 minutes
model_error       10 minutes
unknown_error     2 minutes
```

Every failed probe applies exponential backoff:

```text
next_delay = min(base_delay * 2 ^ probe_failure_count, max_backoff)
```

## Data Model

The module can use SQL for durable account health and Redis for runtime
counters. If the host already has an account table, these fields can be added to
that table or stored in a separate `account_health` table.

### SQL Table

```sql
CREATE TABLE account_health (
  account_id VARCHAR(128) PRIMARY KEY,
  platform VARCHAR(64) NOT NULL,
  group_id VARCHAR(128) NULL,
  health_status VARCHAR(32) NOT NULL DEFAULT 'healthy',
  dispatch_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  priority INTEGER NOT NULL DEFAULT 100,
  weight INTEGER NOT NULL DEFAULT 100,
  max_concurrency INTEGER NOT NULL DEFAULT 1,
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  consecutive_successes INTEGER NOT NULL DEFAULT 0,
  probe_failure_count INTEGER NOT NULL DEFAULT 0,
  last_error_type VARCHAR(64) NULL,
  last_error_code VARCHAR(64) NULL,
  last_error_message TEXT NULL,
  last_success_at TIMESTAMP NULL,
  last_failed_at TIMESTAMP NULL,
  disabled_reason VARCHAR(128) NULL,
  disabled_at TIMESTAMP NULL,
  next_probe_at TIMESTAMP NULL,
  suspect_until TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Optional per-model disable table:

```sql
CREATE TABLE account_model_health (
  account_id VARCHAR(128) NOT NULL,
  model VARCHAR(128) NOT NULL,
  disabled_until TIMESTAMP NULL,
  last_error_type VARCHAR(64) NULL,
  last_error_message TEXT NULL,
  PRIMARY KEY (account_id, model)
);
```

### Redis Keys

```text
aad:account:{account_id}:inflight      ZSET request_id -> deadline_ms
aad:account:{account_id}:rpm           ZSET request_id -> timestamp_ms
aad:account:{account_id}:tpm           ZSET request_id -> timestamp_ms
aad:sticky:{platform}:{session_key}    STRING account_id, TTL
aad:probe:lock:{account_id}            STRING lock, short TTL
aad:decision:{request_id}              HASH selected/skipped/reason, short TTL
```

## Public API

The module can be exposed as HTTP, gRPC, or an in-process package. HTTP is the
lowest-friction option.

### Select Account

`POST /v1/dispatch/select`

Request:

```json
{
  "request_id": "req_123",
  "platform": "openai",
  "group_id": "default",
  "model": "gpt-4o-mini",
  "session_key": "user_or_conversation_id",
  "require_sticky": false,
  "estimated_input_tokens": 1000,
  "estimated_output_tokens": 1000
}
```

Response:

```json
{
  "account_id": "acct_1",
  "health_status": "healthy",
  "decision": "selected",
  "reason": "highest_score",
  "lease_id": "lease_req_123_acct_1",
  "lease_ttl_seconds": 600
}
```

No account response:

```json
{
  "decision": "no_account",
  "reason": "all_accounts_disabled",
  "details": {
    "disabled": 5,
    "rate_limited": 2,
    "concurrency_full": 3
  }
}
```

### Report Success

`POST /v1/dispatch/report-success`

```json
{
  "request_id": "req_123",
  "lease_id": "lease_req_123_acct_1",
  "account_id": "acct_1",
  "platform": "openai",
  "model": "gpt-4o-mini",
  "input_tokens": 1000,
  "output_tokens": 1000,
  "latency_ms": 2300
}
```

Effects:

```text
release concurrency lease
consecutive_failures = 0
last_success_at = now
if health_status == suspect and suspect_until <= now:
  health_status = healthy
```

### Report Failure

`POST /v1/dispatch/report-failure`

```json
{
  "request_id": "req_123",
  "lease_id": "lease_req_123_acct_1",
  "account_id": "acct_1",
  "platform": "openai",
  "model": "gpt-4o-mini",
  "error_type": "rate_limited",
  "error_code": "429",
  "error_message": "too many requests"
}
```

Effects:

```text
release concurrency lease
consecutive_failures += 1
consecutive_successes = 0
last_failed_at = now
if failure threshold reached:
  dispatch_enabled = false
  health_status = disabled
  disabled_reason = error_type
  disabled_at = now
  next_probe_at = now + probe_delay(error_type)
```

### Manual Probe

`POST /v1/dispatch/probe`

```json
{
  "account_id": "acct_1",
  "platform": "openai",
  "model": "gpt-4o-mini"
}
```

This endpoint is useful for an admin UI button named "Check now".

### Get Account Health

`GET /v1/dispatch/accounts/{account_id}/health`

Response:

```json
{
  "account_id": "acct_1",
  "health_status": "disabled",
  "dispatch_enabled": false,
  "consecutive_failures": 2,
  "consecutive_successes": 0,
  "last_error_type": "rate_limited",
  "disabled_reason": "rate_limited",
  "disabled_at": "2026-06-03T13:00:00Z",
  "next_probe_at": "2026-06-03T13:05:00Z"
}
```

## Selection Algorithm

Candidate filters:

```text
platform matches
group_id matches if provided
dispatch_enabled = true
health_status in (healthy, suspect)
model is not disabled for the account
current_concurrency < max_concurrency
rpm/tpm windows are not exceeded
```

Score:

```text
score =
  priority * 10
  + weight
  + sticky_bonus
  - concurrency_penalty
  - suspect_penalty
  - recent_error_penalty
```

Suggested values:

```text
sticky_bonus          1000
suspect_penalty       300
recent_error_penalty  consecutive_failures * 100
concurrency_penalty   current_inflight / max_concurrency * 500
```

Concurrency lease must be atomic. Use a Redis Lua script or DB row lock to avoid
multiple gateway instances selecting the same final slot at the same time.

## Health Checker Loop

The health checker runs continuously.

```pseudo
every checker_scan_interval:
  accounts = find accounts where:
    next_probe_at <= now
    and health_status in (disabled, suspect, healthy)

  for account in accounts:
    if not acquire_probe_lock(account.id):
      continue

    set health_status = probing if account.dispatch_enabled == false

    result = probe(account)

    if result.success:
      consecutive_successes += 1
      probe_failure_count = 0
      last_success_at = now

      if consecutive_successes >= recovery_success_threshold:
        dispatch_enabled = true
        health_status = suspect
        suspect_until = now + suspect_window
        consecutive_failures = 0
        next_probe_at = now + suspect_probe_interval
      else:
        next_probe_at = now + probe_delay(last_error_type)

    else:
      consecutive_failures += 1
      consecutive_successes = 0
      probe_failure_count += 1
      dispatch_enabled = false
      health_status = disabled
      last_error_type = result.error_type
      disabled_reason = result.error_type
      next_probe_at = now + backoff(result.error_type, probe_failure_count)

    release_probe_lock(account.id)
```

When a suspect account receives real successful traffic and `suspect_until` has
passed, promote it to `healthy`.

## Probe Implementation

Each platform needs a small adapter:

```text
OpenAI-compatible:
  preferred: GET /v1/models
  stronger: POST /v1/chat/completions with max_tokens=1

Claude-compatible:
  POST /v1/messages with max_tokens=1

Gemini-compatible:
  GET models list or minimal generate request

Custom:
  configured HTTP method, URL, headers, body, success matcher
```

Use a real minimal model request if account validity depends on model access.
Use `models` list only if it is known to catch disabled or expired accounts.

## Host Adapter Interface

If implemented as an in-process package, expose these interfaces:

```go
type AccountStore interface {
    ListCandidates(ctx context.Context, req SelectRequest) ([]Account, error)
    UpdateHealth(ctx context.Context, update HealthUpdate) error
    GetProbeCredentials(ctx context.Context, accountID string) (ProbeCredentials, error)
}

type RuntimeStore interface {
    TryAcquireLease(ctx context.Context, accountID string, requestID string, ttl time.Duration) (Lease, bool, error)
    ReleaseLease(ctx context.Context, leaseID string) error
    GetInflight(ctx context.Context, accountID string) (int, error)
    SetSticky(ctx context.Context, key string, accountID string, ttl time.Duration) error
    GetSticky(ctx context.Context, key string) (string, error)
}

type ProbeAdapter interface {
    Probe(ctx context.Context, account Account, model string) ProbeResult
}
```

## Admin UI Fields

Add these columns to the account list:

```text
Dispatch enabled
Health status
Consecutive failures
Consecutive successes
Last error type
Disabled reason
Last success time
Last failed time
Next probe time
Manual check button
Manual enable/disable button
```

## Safe Rollout

1. Add the module in observe-only mode.
2. Log which accounts would be disabled, but do not disable them.
3. Enable automatic disable for `auth_error` and `quota_exhausted`.
4. Enable automatic disable for `rate_limited`, `timeout`, and `server_error`.
5. Enable automatic recovery probing.
6. Enable suspect-state weighted dispatch.

## Minimal First Version

Implement only:

```text
select account by dispatch_enabled + concurrency
report_success resets failures
report_failure increments failures
disable after threshold
checker scans every 30 seconds
disabled accounts probe after next_probe_at
re-enable after 2 consecutive probe successes
```

This version is enough to automatically close bad accounts and reopen recovered
accounts without changing the host gateway architecture.

