package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Sub2AdminProbeAdapter struct {
	client       *http.Client
	baseURL      string
	email        string
	password     string
	mu           sync.Mutex
	accessToken  string
	tokenExpires time.Time
}

func NewSub2AdminProbeAdapter(baseURL string, email string, password string, timeout time.Duration) (*Sub2AdminProbeAdapter, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" || email == "" || password == "" {
		return nil, errors.New("sub2 admin probe requires base url, email, and password")
	}
	return &Sub2AdminProbeAdapter{
		client:   &http.Client{Timeout: timeout},
		baseURL:  baseURL,
		email:    email,
		password: password,
	}, nil
}

func (p *Sub2AdminProbeAdapter) Probe(ctx context.Context, account Account, model string) ProbeResult {
	token, err := p.token(ctx)
	trace := &ProbeHTTPTrace{
		Method: http.MethodPost,
		URL:    RedactSensitiveURL(p.baseURL + "/admin/accounts/" + account.AccountID + "/test"),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "[REDACTED]",
		},
		Body: "{}",
	}
	if err != nil {
		return ProbeResult{Success: false, ErrorType: ErrAuthError, ErrorMessage: "sub2 admin login failed: " + err.Error(), Request: trace}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/admin/accounts/"+account.AccountID+"/test", bytes.NewBufferString("{}"))
	if err != nil {
		return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: err.Error(), Request: trace}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		result := classifyProbeNetworkError(err)
		result.Request = trace
		return result
	}
	defer resp.Body.Close()

	responseTrace := &ProbeHTTPTrace{Status: resp.Status, StatusCode: resp.StatusCode, Headers: safeResponseHeaders(resp.Header)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		errorType, message := classifyHTTPStatus(resp.StatusCode)
		responseTrace.BodySample = RedactSensitiveText(truncateTrace(string(body), 4000))
		return ProbeResult{Success: false, ErrorType: errorType, ErrorCode: resp.Status, ErrorMessage: "sub2 account test failed: " + message, Request: trace, Response: responseTrace}
	}

	result := parseSub2AccountTestStream(resp.Body)
	result.Request = trace
	if result.Response == nil {
		result.Response = responseTrace
	} else {
		result.Response.Status = responseTrace.Status
		result.Response.StatusCode = responseTrace.StatusCode
		result.Response.Headers = responseTrace.Headers
	}
	return result
}

func (p *Sub2AdminProbeAdapter) token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.tokenExpires) {
		return p.accessToken, nil
	}

	payload, _ := json.Marshal(map[string]string{
		"email":    p.email,
		"password": p.password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", errors.New(resp.Status + ": " + string(body))
	}

	var login struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&login); err != nil {
		return "", err
	}
	if login.Code != 0 || login.Data.AccessToken == "" {
		if login.Message == "" {
			login.Message = "missing access token"
		}
		return "", errors.New(login.Message)
	}

	expiresIn := login.Data.ExpiresIn
	if expiresIn <= 120 {
		expiresIn = 3600
	}
	p.accessToken = login.Data.AccessToken
	p.tokenExpires = time.Now().Add(time.Duration(expiresIn-60) * time.Second)
	return p.accessToken, nil
}

func parseSub2AccountTestStream(body io.Reader) ProbeResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	var contentSeen bool
	var lastMessage string
	var lastEvent string
	var bodySample strings.Builder
	events := make([]ProbeEventTrace, 0, 12)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		appendTraceSample(&bodySample, line)
		if strings.HasPrefix(line, "event:") {
			lastEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "" || raw == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			lastMessage = RedactSensitiveText(raw)
			events = appendEventTrace(events, ProbeEventTrace{Event: lastEvent, Data: RedactSensitiveText(truncateTrace(raw, 1500))})
			continue
		}

		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = lastEvent
		}
		events = appendEventTrace(events, ProbeEventTrace{Event: lastEvent, Type: eventType, Data: RedactSensitiveText(truncateTrace(raw, 1500))})
		switch eventType {
		case "content":
			contentSeen = true
		case "response.failed":
			return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: eventMessage(event, "sub2 account test response failed"), Response: streamTrace(bodySample.String()), Events: events}
		case "test_complete":
			if success, _ := event["success"].(bool); success {
				return ProbeResult{Success: true, Response: streamTrace(bodySample.String()), Events: events}
			}
			return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: eventMessage(event, "sub2 account test completed with failure"), Response: streamTrace(bodySample.String()), Events: events}
		case "error", "test_error":
			return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: eventMessage(event, "sub2 account test error"), Response: streamTrace(bodySample.String()), Events: events}
		default:
			if msg := eventMessage(event, ""); msg != "" {
				lastMessage = RedactSensitiveText(msg)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		result := classifyProbeNetworkError(err)
		result.Response = streamTrace(bodySample.String())
		result.Events = events
		return result
	}
	if contentSeen {
		return ProbeResult{Success: true, Response: streamTrace(bodySample.String()), Events: events}
	}
	if lastMessage != "" {
		return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: lastMessage, Response: streamTrace(bodySample.String()), Events: events}
	}
	return ProbeResult{Success: false, ErrorType: ErrUnknown, ErrorMessage: "sub2 account test returned no completion event", Response: streamTrace(bodySample.String()), Events: events}
}

func eventMessage(event map[string]any, fallback string) string {
	for _, key := range []string{"message", "error", "detail", "text"} {
		if value, ok := event[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	for _, key := range []string{"error", "response"} {
		nested, ok := event[key].(map[string]any)
		if !ok {
			continue
		}
		if message := eventMessage(nested, ""); message != "" {
			return message
		}
	}
	return fallback
}

func classifyProbeNetworkError(err error) ProbeResult {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "timeout") || strings.Contains(message, "deadline") {
		return ProbeResult{Success: false, ErrorType: ErrTimeout, ErrorMessage: err.Error()}
	}
	return ProbeResult{Success: false, ErrorType: ErrNetworkError, ErrorMessage: err.Error()}
}

func streamTrace(sample string) *ProbeHTTPTrace {
	return &ProbeHTTPTrace{BodySample: RedactSensitiveText(truncateTrace(sample, 4000))}
}

func appendTraceSample(builder *strings.Builder, line string) {
	if builder.Len() >= 4000 {
		return
	}
	if builder.Len() > 0 {
		builder.WriteByte('\n')
	}
	builder.WriteString(line)
}

func appendEventTrace(events []ProbeEventTrace, event ProbeEventTrace) []ProbeEventTrace {
	if len(events) >= 20 {
		return events
	}
	return append(events, event)
}

func safeResponseHeaders(header http.Header) map[string]string {
	headers := map[string]string{}
	for key, values := range header {
		headers[key] = RedactSensitiveHeaders(map[string]string{key: truncateTrace(strings.Join(values, ", "), 500)})[key]
	}
	return headers
}

func truncateTrace(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...[truncated]"
}
