package core

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

type HTTPProbeAdapter struct {
	client *http.Client
}

func NewHTTPProbeAdapter(timeout time.Duration) *HTTPProbeAdapter {
	return &HTTPProbeAdapter{client: &http.Client{Timeout: timeout}}
}

func (p *HTTPProbeAdapter) Probe(ctx context.Context, account Account, model string) ProbeResult {
	if account.ProbeURL == "" {
		return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: "missing auto probe config"}
	}

	method := account.ProbeMethod
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if account.ProbeBody != "" {
		body = bytes.NewBufferString(account.ProbeBody)
	}
	trace := &ProbeHTTPTrace{
		Method:  method,
		URL:     RedactSensitiveURL(account.ProbeURL),
		Headers: RedactSensitiveHeaders(account.ProbeHeaders),
		Body:    RedactSensitiveText(truncateTrace(account.ProbeBody, 4000)),
	}

	req, err := http.NewRequestWithContext(ctx, method, account.ProbeURL, body)
	if err != nil {
		return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: err.Error(), Request: trace}
	}
	for key, value := range account.ProbeHeaders {
		req.Header.Set(key, value)
	}
	if account.ProbeBody != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result := classifyProbeNetworkError(err)
		result.Request = trace
		return result
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	responseTrace := &ProbeHTTPTrace{Status: resp.Status, StatusCode: resp.StatusCode, Headers: safeResponseHeaders(resp.Header)}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ProbeResult{Success: true, Request: trace, Response: responseTrace}
	}
	errorType, message := classifyHTTPStatus(resp.StatusCode)
	return ProbeResult{Success: false, ErrorType: errorType, ErrorCode: resp.Status, ErrorMessage: message, Request: trace, Response: responseTrace}
}

func classifyHTTPStatus(status int) (string, string) {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrAuthError, "authentication failed: upstream returned " + http.StatusText(status)
	case status == http.StatusTooManyRequests:
		return ErrRateLimited, "rate limited: upstream returned 429"
	case status == 529:
		return ErrOverloaded, "upstream overloaded: returned 529"
	case status >= 500:
		return ErrServerError, "upstream server error: " + http.StatusText(status)
	default:
		return ErrUnknown, "unexpected upstream status: " + http.StatusText(status)
	}
}
