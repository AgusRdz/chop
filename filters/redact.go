package filters

import (
	"regexp"
	"strings"
)

// sensitiveKeywords is a list of keys and keywords that often contain secrets.
var sensitiveKeywords = []string{
	"authorization",
	"cookie",
	"set-cookie",
	"x-auth-token",
	"x-api-key",
	"api-key",
	"api_key",
	"proxy-authorization",
	"password",
	"secret",
	"token",
	"apikey",
	"access-token",
	"session",
	"x-csrf-token",
	"private_key",
	"client_secret",
	"db_password",
}

// sensitiveHeadersRe matches HTTP header lines containing any sensitive keyword
// anywhere in the header name (e.g., "X-Custom-Token:", "Refresh-Token:").
var sensitiveHeadersRe = regexp.MustCompile(`(?mi)^([<>* ]*)([^\s:]*(?:` + strings.Join(sensitiveKeywords, "|") + `)[^\s:]*):.*$`)

// redactHeaders masks sensitive HTTP headers and key-value pairs in plain text.
func redactHeaders(s string) string {
	return sensitiveHeadersRe.ReplaceAllString(s, "${1}${2}: [REDACTED]")
}

// RedactOutput applies header-level secret redaction to arbitrary command output.
// Used on the failure path where no filter runs but output still contains secrets.
func RedactOutput(s string) string {
	return redactHeaders(s)
}

// redactJSON recursively redacts sensitive keys in parsed JSON data.
func redactJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, v := range val {
			if isSensitiveKey(k) {
				newMap[k] = "[REDACTED]"
			} else {
				// Special handling for common environment variable lists (e.g., Docker inspect)
				if strings.EqualFold(k, "env") || strings.EqualFold(k, "environment") {
					newMap[k] = redactEnvList(v)
				} else {
					newMap[k] = redactJSON(v)
				}
			}
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, v := range val {
			newSlice[i] = redactJSON(v)
		}
		return newSlice
	default:
		return v
	}
}

// isSensitiveKey returns true if the key matches any of our sensitive keywords.
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, kw := range sensitiveKeywords {
		if strings.Contains(lowerKey, kw) {
			return true
		}
	}
	return false
}

// redactEnvList handles lists of "KEY=VALUE" strings, redacting the value if the key is sensitive.
func redactEnvList(v interface{}) interface{} {
	slice, ok := v.([]interface{})
	if !ok {
		return redactJSON(v)
	}

	newSlice := make([]interface{}, len(slice))
	for i, item := range slice {
		str, ok := item.(string)
		if !ok {
			newSlice[i] = redactJSON(item)
			continue
		}

		parts := strings.SplitN(str, "=", 2)
		if len(parts) == 2 && isSensitiveKey(parts[0]) {
			newSlice[i] = parts[0] + "=[REDACTED]"
		} else {
			newSlice[i] = str
		}
	}
	return newSlice
}
