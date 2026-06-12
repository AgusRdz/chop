package filters

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func filterKubectlLogs(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var result string
	// Detect if structured JSON logs
	if isJSONLogs(lines) {
		result = filterJSONLogs(lines)
	} else {
		result = filterTextLogs(lines)
	}

	return outputSanityCheck(raw, result), nil
}

func isJSONLogs(lines []string) bool {
	checked := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "{") {
			return false
		}
		checked++
		if checked >= 3 {
			return true
		}
	}
	return checked > 0
}

type kubectlLogEntry struct {
	original  string
	timestamp string
	level     string
	message   string
	isError   bool
}

type kubectlDedupEntry struct {
	entry kubectlLogEntry
	count int
}

func filterJSONLogs(lines []string) string {
	var entries []kubectlLogEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			entries = append(entries, kubectlLogEntry{original: line})
			continue
		}

		ts := jsonStr(obj, "timestamp", "time", "ts", "@timestamp")
		level := strings.ToUpper(jsonStr(obj, "level", "severity", "lvl"))
		msg := jsonStr(obj, "message", "msg", "error")

		isErr := level == "ERROR" || level == "WARN" || level == "WARNING" || level == "FATAL"

		entries = append(entries, kubectlLogEntry{
			timestamp: ts,
			level:     level,
			message:   msg,
			isError:   isErr,
		})
	}

	return deduplicateLogEntries(entries)
}

func deduplicateLogEntries(entries []kubectlLogEntry) string {
	var deduped []kubectlDedupEntry
	for _, e := range entries {
		key := entryKey(e)
		if len(deduped) > 0 {
			last := &deduped[len(deduped)-1]
			if entryKey(last.entry) == key {
				last.count++
				continue
			}
		}
		deduped = append(deduped, kubectlDedupEntry{entry: e, count: 1})
	}

	var errorEntries []kubectlDedupEntry
	var normalEntries []kubectlDedupEntry
	for _, d := range deduped {
		if d.entry.isError {
			errorEntries = append(errorEntries, d)
		} else {
			normalEntries = append(normalEntries, d)
		}
	}

	hidden := 0
	if len(normalEntries) > 50 {
		for _, d := range normalEntries[:len(normalEntries)-50] {
			hidden += d.count
		}
		normalEntries = normalEntries[len(normalEntries)-50:]
	}

	var out []string
	if hidden > 0 {
		out = append(out, fmt.Sprintf("(%d earlier lines hidden)", hidden))
	}

	for _, d := range errorEntries {
		out = append(out, formatDedupEntry(d))
	}
	for _, d := range normalEntries {
		out = append(out, formatDedupEntry(d))
	}

	return strings.Join(out, "\n")
}

func entryKey(e kubectlLogEntry) string {
	if e.original != "" {
		return e.original
	}
	return e.level + "|" + e.message
}

func formatDedupEntry(d kubectlDedupEntry) string {
	var sb strings.Builder
	if d.entry.original != "" {
		sb.WriteString(d.entry.original)
	} else {
		first := true
		if d.entry.timestamp != "" {
			sb.WriteString(d.entry.timestamp)
			first = false
		}
		if d.entry.level != "" {
			if !first {
				sb.WriteByte(' ')
			}
			sb.WriteString(d.entry.level)
			first = false
		}
		if d.entry.message != "" {
			if !first {
				sb.WriteByte(' ')
			}
			sb.WriteString(d.entry.message)
		}
	}
	if d.count > 1 {
		sb.WriteString(" (x")
		sb.WriteString(strconv.Itoa(d.count))
		sb.WriteByte(')')
	}
	return sb.String()
}

func filterTextLogs(lines []string) string {
	// Strip DEBUG/TRACE and collect cleaned lines
	var cleaned []string
	totalLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		totalLines++
		cleaned = append(cleaned, trimmed)
	}

	stripDebug := totalLines > 100
	if stripDebug {
		var kept []string
		for _, line := range cleaned {
			if isDebugLine(line) && !isErrorLine(line) {
				continue
			}
			kept = append(kept, line)
		}
		cleaned = kept
	}

	// Try pattern-based compression first
	if result, ok := compressLogPatterns(cleaned, isErrorLine); ok {
		return result
	}

	// Fallback: consecutive exact-match dedup
	type dedupLine struct {
		line    string
		count   int
		isError bool
	}

	var deduped []dedupLine
	for _, line := range cleaned {
		isErr := isErrorLine(line)
		if len(deduped) > 0 && deduped[len(deduped)-1].line == line {
			deduped[len(deduped)-1].count++
			continue
		}
		deduped = append(deduped, dedupLine{line: line, count: 1, isError: isErr})
	}

	var errorLines []dedupLine
	var normalLines []dedupLine
	for _, d := range deduped {
		if d.isError {
			errorLines = append(errorLines, d)
		} else {
			normalLines = append(normalLines, d)
		}
	}

	hidden := 0
	if len(normalLines) > 50 {
		for _, d := range normalLines[:len(normalLines)-50] {
			hidden += d.count
		}
		normalLines = normalLines[len(normalLines)-50:]
	}

	var out []string
	if hidden > 0 {
		out = append(out, fmt.Sprintf("(%d earlier lines hidden)", hidden))
	}

	for _, d := range errorLines {
		line := d.line
		if d.count > 1 {
			line += " (x" + strconv.Itoa(d.count) + ")"
		}
		out = append(out, line)
	}

	for _, d := range normalLines {
		line := d.line
		if d.count > 1 {
			line += " (x" + strconv.Itoa(d.count) + ")"
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func isErrorLine(line string) bool {
	upper := strings.ToUpper(line)
	return strings.Contains(upper, "ERROR") ||
		strings.Contains(upper, "WARN") ||
		strings.Contains(upper, "FATAL") ||
		strings.Contains(upper, "PANIC")
}

func isDebugLine(line string) bool {
	upper := strings.ToUpper(line)
	return strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE")
}

func jsonStr(obj map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			switch vv := v.(type) {
			case string:
				return vv
			case float64:
				return fmt.Sprintf("%.0f", vv)
			default:
				return fmt.Sprintf("%v", vv)
			}
		}
	}
	return ""
}
