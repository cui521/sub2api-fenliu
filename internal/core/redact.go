package core

import (
	"net/url"
	"regexp"
	"strings"
)

type redactionRule struct {
	pattern     *regexp.Regexp
	replacement string
}

var sensitiveValueRules = []redactionRule{
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)((?:api[_-]?key|access[_-]?token|refresh[_-]?token|token|password|passwd|secret|authorization)["']?\s*[:=]\s*["']?)[^"',\s}\\]+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)sk-[A-Za-z0-9._-]{8,}`), `[REDACTED]`},
}

func RedactSensitiveText(value string) string {
	if value == "" {
		return value
	}
	redacted := value
	for _, rule := range sensitiveValueRules {
		redacted = rule.pattern.ReplaceAllString(redacted, rule.replacement)
	}
	return redacted
}

func RedactSensitiveURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return RedactSensitiveText(rawURL)
	}
	query := parsed.Query()
	for key := range query {
		if isSensitiveName(key) {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return RedactSensitiveText(parsed.String())
}

func RedactSensitiveHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	safe := make(map[string]string, len(headers))
	for key, value := range headers {
		if isSensitiveName(key) {
			safe[key] = "[REDACTED]"
			continue
		}
		safe[key] = RedactSensitiveText(value)
	}
	return safe
}

func RedactProbeResult(result ProbeResult) ProbeResult {
	result.ErrorCode = RedactSensitiveText(result.ErrorCode)
	result.ErrorMessage = RedactSensitiveText(result.ErrorMessage)
	result.Request = RedactProbeHTTPTrace(result.Request)
	result.Response = RedactProbeHTTPTrace(result.Response)
	for i := range result.Events {
		result.Events[i].Data = RedactSensitiveText(result.Events[i].Data)
	}
	return result
}

func RedactProbeHTTPTrace(trace *ProbeHTTPTrace) *ProbeHTTPTrace {
	if trace == nil {
		return nil
	}
	copy := *trace
	copy.URL = RedactSensitiveURL(copy.URL)
	copy.Headers = RedactSensitiveHeaders(copy.Headers)
	copy.Body = RedactSensitiveText(copy.Body)
	copy.BodySample = RedactSensitiveText(copy.BodySample)
	copy.Status = RedactSensitiveText(copy.Status)
	return &copy
}

func isSensitiveName(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{"authorization", "cookie", "key", "token", "password", "passwd", "secret"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
