package filters

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// filterCurl filters curl command output, compressing JSON responses
// and summarizing HTML/binary content.
func filterCurl(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	// Check for curl errors (connection refused, timeout, etc.)
	if isCurlError(raw) {
		return raw, nil
	}

	// Check for HTTP headers (curl -i, -v, --include)
	body, statusLine := extractHTTPBody(raw)

	// Determine content type and filter accordingly
	trimmedBody := strings.TrimSpace(body)

	// JSON response
	if isJSON(trimmedBody) {
		compressed, err := compressJSON(trimmedBody)
		if err != nil {
			// Fallback: return raw if compression fails
			return raw, nil
		}
		if statusLine != "" {
			return statusLine + "\n" + compressed, nil
		}
		return compressed, nil
	}

	// HTML response
	if isHTML(trimmedBody) {
		size := len(trimmedBody)
		summary := fmt.Sprintf("HTML response (%d bytes)", size)
		if statusLine != "" {
			return statusLine + "\n" + summary, nil
		}
		return summary, nil
	}

	// Binary detection
	if isBinary(trimmedBody) {
		size := len(trimmedBody)
		summary := fmt.Sprintf("binary response (%d bytes)", size)
		if statusLine != "" {
			return statusLine + "\n" + summary, nil
		}
		return summary, nil
	}

	// Plain text: pass through if <100 lines, truncate otherwise
	lines := strings.Split(trimmedBody, "\n")
	if len(lines) > 100 {
		truncated := strings.Join(lines[:100], "\n")
		truncated += fmt.Sprintf("\n... (%d more lines)", len(lines)-100)
		if statusLine != "" {
			return statusLine + "\n" + truncated, nil
		}
		return truncated, nil
	}

	if statusLine != "" {
		result := statusLine + "\n" + trimmedBody
		return outputSanityCheck(raw, result), nil
	}
	return outputSanityCheck(raw, trimmedBody), nil
}

// isCurlError detects curl error messages.
func isCurlError(raw string) bool {
	errorPrefixes := []string{
		"curl:",
		"curl: (",
	}
	lower := strings.ToLower(raw)
	for _, prefix := range errorPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	// Also catch "Could not resolve host", "Connection refused", etc.
	curlErrors := []string{
		"could not resolve host",
		"connection refused",
		"connection timed out",
		"operation timed out",
		"ssl certificate problem",
		"failed to connect",
	}
	for _, msg := range curlErrors {
		if strings.Contains(lower, msg) {
			return true
		}
	}
	return false
}

// extractHTTPBody separates HTTP headers from body in curl -i/-v output.
// Returns the body and the status line (if headers were present).
func extractHTTPBody(raw string) (body string, statusLine string) {
	// Look for HTTP status line
	if !strings.HasPrefix(raw, "HTTP/") {
		return raw, ""
	}

	// Find the blank line that separates headers from body
	// HTTP headers end with \r\n\r\n or \n\n
	separators := []string{"\r\n\r\n", "\n\n"}
	for _, sep := range separators {
		idx := strings.Index(raw, sep)
		if idx != -1 {
			headerBlock := raw[:idx]
			body = raw[idx+len(sep):]

			// Extract status line (first line)
			lines := strings.SplitN(headerBlock, "\n", 2)
			statusLine = strings.TrimSpace(lines[0])

			return body, statusLine
		}
	}

	// No blank line separator found - might be headers only
	lines := strings.SplitN(raw, "\n", 2)
	return raw, strings.TrimSpace(lines[0])
}

// isJSON checks if the string looks like JSON.
func isJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

// isHTML checks if the string looks like HTML.
func isHTML(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(lower, "<!doctype html") ||
		strings.HasPrefix(lower, "<html") ||
		strings.HasPrefix(lower, "<!doctype") ||
		(strings.Contains(lower, "<head>") && strings.Contains(lower, "<body"))
}

// isBinary checks if the string contains non-printable characters suggesting binary data.
func isBinary(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check first 512 bytes for non-UTF8 or control chars
	sample := s
	if len(sample) > 512 {
		sample = sample[:512]
	}
	nonPrintable := 0
	total := 0
	for i := 0; i < len(sample); {
		r, size := utf8.DecodeRuneInString(sample[i:])
		if r == utf8.RuneError && size == 1 {
			nonPrintable++
		} else if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			nonPrintable++
		}
		total++
		i += size
	}
	// If >10% non-printable, treat as binary
	return total > 0 && float64(nonPrintable)/float64(total) > 0.1
}
