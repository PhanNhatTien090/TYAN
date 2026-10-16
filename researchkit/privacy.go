package researchkit

import (
	"encoding/json"
	"net/url"
	"regexp"
)

var sensitiveKey = regexp.MustCompile(`(?i)(password|passwd|secret|token|authorization|cookie|api[-_]?key|private[-_]?key|email|phone|address|national[-_]?id)`)
var headerSecret = regexp.MustCompile(`(?im)^(authorization|proxy-authorization|cookie|set-cookie|x-api-key|api-key)\s*:[^\r\n]*`)
var bearerSecret = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/-]+=*`)
var emailPII = regexp.MustCompile(`\b[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
var assignmentSecret = regexp.MustCompile(`(?i)\b(password|passwd|secret|access_token|refresh_token|api_key|apikey)\s*([=:])\s*([^\s&;,]+)`)

type Redaction struct {
	Text   string `json:"text"`
	Notice string `json:"notice"`
}

func Redact(input string) (Redaction, error) {
	if err := CheckInput(input); err != nil {
		return Redaction{}, err
	}
	text := input
	var doc any
	if json.Unmarshal([]byte(input), &doc) == nil {
		doc = redactValue(doc)
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return Redaction{}, err
		}
		text = string(b)
	}
	text = headerSecret.ReplaceAllString(text, "$1: [REDACTED]")
	text = bearerSecret.ReplaceAllString(text, "Bearer [REDACTED]")
	text = assignmentSecret.ReplaceAllString(text, "$1$2[REDACTED]")
	text = emailPII.ReplaceAllString(text, "[REDACTED_EMAIL]")
	return Redaction{Text: text, Notice: "Best-effort redaction only. Manually inspect free text, URL paths, screenshots, encoded values and files before sharing."}, nil
}

func redactValue(v any) any {
	switch value := v.(type) {
	case map[string]any:
		for key, item := range value {
			if sensitiveKey.MatchString(key) {
				value[key] = "[REDACTED]"
			} else {
				value[key] = redactValue(item)
			}
		}
	case []any:
		for i, item := range value {
			value[i] = redactValue(item)
		}
	}
	return v
}

func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid URL]"
	}
	u.User = nil
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		q.Set(key, "[REDACTED]")
	}
	u.RawQuery = q.Encode()
	return u.String()
}
