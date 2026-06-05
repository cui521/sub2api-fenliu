# Account Auto Dispatch

Standalone Go service for automatic account dispatch, health detection, disable,
probe, and recovery.

This first version uses only the Go standard library:

- HTTP API
- JSON file persistence
- in-memory runtime leases
- background health checker

SQLite and Redis are intentionally hidden behind interfaces so they can be
added without changing the API or scheduler logic.

## Run

```powershell
go run ./cmd/aad
```

Default server:

```text
http://127.0.0.1:18080
```

Default data file:

```text
./data/accounts.json
```

## Access control

Set `AAD_WEB_TOKEN` before exposing the service outside localhost. Admin pages and
admin APIs require this token through the login cookie or `X-AAD-Token` header.

The public read-only status page does not require the token:

```text
GET /status
GET /v1/public/status
```

The public status API only returns each station display name and coarse status.
It does not expose account IDs, keys, probe logs, errors, or control actions.

If `AAD_WEB_TOKEN` is missing, admin access is locked by default. For local-only
debugging you can explicitly set `AAD_ALLOW_UNSAFE_NO_AUTH=true`.

## APIs

```text
GET  /status
POST /v1/accounts
GET  /v1/accounts
GET  /v1/dispatch/accounts/{account_id}/health
POST /v1/dispatch/select
POST /v1/dispatch/report-success
POST /v1/dispatch/report-failure
POST /v1/dispatch/probe
```

## Example

Create account:

```powershell
curl.exe -X POST http://127.0.0.1:18080/v1/accounts `
  -H "Content-Type: application/json" `
  -d "{\"account_id\":\"acct_1\",\"platform\":\"openai\",\"group_id\":\"default\",\"priority\":100,\"weight\":100,\"max_concurrency\":2,\"probe_url\":\"https://api.openai.com/v1/models\",\"probe_method\":\"GET\",\"probe_headers\":{\"Authorization\":\"Bearer sk-xxx\"}}"
```

Select account:

```powershell
curl.exe -X POST http://127.0.0.1:18080/v1/dispatch/select `
  -H "Content-Type: application/json" `
  -d "{\"request_id\":\"req_1\",\"platform\":\"openai\",\"group_id\":\"default\",\"model\":\"gpt-4o-mini\",\"session_key\":\"user_1\"}"
```

Report failure:

```powershell
curl.exe -X POST http://127.0.0.1:18080/v1/dispatch/report-failure `
  -H "Content-Type: application/json" `
  -d "{\"request_id\":\"req_1\",\"lease_id\":\"lease_req_1_acct_1\",\"account_id\":\"acct_1\",\"platform\":\"openai\",\"model\":\"gpt-4o-mini\",\"error_type\":\"rate_limited\",\"error_code\":\"429\",\"error_message\":\"too many requests\"}"
```
