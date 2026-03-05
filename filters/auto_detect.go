package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var logPattern = regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}|\d{2}:\d{2}:\d{2}|\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`)
var logLevelPattern = regexp.MustCompile(`(?i)\b(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE|CRITICAL)\b`)

// AutoDetect applies generic format-aware compression to any command output.
// Returns the raw output unchanged if it's already short or uncompressible.
func AutoDetect(raw string) (string, error) {
	return filterAutoDetect(raw)
}

func filterAutoDetect(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	trimmed := strings.TrimSpace(raw)
	lines := strings.Split(trimmed, "\n")

	// Don't compress short output
	if len(lines) < 20 && len(trimmed) < 500 {
		return raw, nil
	}

	// JSON detection
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		result, err := compressJSON(trimmed)
		if err == nil {
			return result, nil
		}
		// Not valid JSON, fall through
	}

	// XML/HTML detection
	if len(trimmed) > 0 && trimmed[0] == '<' && looksLikeMarkup(trimmed) {
		return compressMarkup(trimmed, lines), nil
	}

	// CSV/TSV detection
	if looksLikeCSV(lines) {
		return compressCSV(lines), nil
	}

	// Table detection
	if looksLikeTable(lines) {
		return compressTable(lines), nil
	}

	// Log-like detection
	if looksLikeLog(lines) {
		return compressLog(lines), nil
	}

	// Plain text fallback
	return compressPlainText(lines), nil
}

// --- CSV/TSV detection and compression ---

func looksLikeCSV(lines []string) bool {
	if len(lines) < 2 {
		return false
	}

	// Detect delimiter: comma or tab
	delim := detectCSVDelimiter(lines)
	if delim == 0 {
		return false
	}

	// Check consistent column count across first 5 lines
	headerCount := strings.Count(lines[0], string(delim)) + 1
	if headerCount < 2 {
		return false
	}

	checkLines := lines[1:]
	if len(checkLines) > 4 {
		checkLines = checkLines[:4]
	}
	for _, line := range checkLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		count := strings.Count(line, string(delim)) + 1
		if count != headerCount {
			return false
		}
	}
	return true
}

func detectCSVDelimiter(lines []string) byte {
	if len(lines) == 0 {
		return 0
	}
	header := lines[0]
	commas := strings.Count(header, ",")
	tabs := strings.Count(header, "\t")
	if commas >= 2 {
		return ','
	}
	if tabs >= 2 {
		return '\t'
	}
	return 0
}

func compressCSV(lines []string) string {
	var b strings.Builder
	delim := detectCSVDelimiter(lines)
	colCount := strings.Count(lines[0], string(delim)) + 1

	// Header
	b.WriteString(lines[0])
	b.WriteString("\n")

	// Data rows (up to 5)
	dataLines := lines[1:]
	showCount := 5
	if len(dataLines) < showCount {
		showCount = len(dataLines)
	}
	for i := 0; i < showCount; i++ {
		b.WriteString(dataLines[i])
		b.WriteString("\n")
	}

	remaining := len(dataLines) - showCount
	if remaining > 0 {
		b.WriteString(fmt.Sprintf("... and %d more rows\n", remaining))
	}
	b.WriteString(fmt.Sprintf("(%d columns)", colCount))

	return b.String()
}

// --- Table detection and compression ---

func looksLikeTable(lines []string) bool {
	if len(lines) < 3 {
		return false
	}

	separatorCount := 0
	pipeCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSeparatorLine(trimmed) {
			separatorCount++
		}
		if strings.Contains(trimmed, "|") {
			pipeCount++
		}
	}

	// Table if has separator lines or many pipe-delimited lines
	if separatorCount >= 1 && pipeCount >= 2 {
		return true
	}

	// Aligned columns: check if multiple spaces appear at consistent positions
	if hasAlignedColumns(lines) {
		return true
	}

	return false
}

func isSeparatorLine(line string) bool {
	if line == "" {
		return false
	}
	cleaned := strings.ReplaceAll(line, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "=", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	cleaned = strings.ReplaceAll(cleaned, "|", "")
	cleaned = strings.TrimSpace(cleaned)
	return len(cleaned) == 0 && len(line) > 3
}

func hasAlignedColumns(lines []string) bool {
	if len(lines) < 5 {
		return false
	}
	// Check for consistent multi-space gaps (2+ spaces) indicating alignment
	multiSpaceCount := 0
	for _, line := range lines[:min(10, len(lines))] {
		if strings.Contains(line, "  ") && len(strings.Fields(line)) >= 3 {
			multiSpaceCount++
		}
	}
	return multiSpaceCount >= len(lines[:min(10, len(lines))])*7/10
}

func compressTable(lines []string) string {
	var b strings.Builder
	var dataRows []string

	// Collect header (first non-separator line) and data rows
	headerFound := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isSeparatorLine(trimmed) {
			if !headerFound {
				continue // skip leading separators
			}
			continue // skip all separators
		}
		if !headerFound {
			b.WriteString(line)
			b.WriteString("\n")
			headerFound = true
			continue
		}
		dataRows = append(dataRows, line)
	}

	showCount := 10
	if len(dataRows) < showCount {
		showCount = len(dataRows)
	}
	for i := 0; i < showCount; i++ {
		b.WriteString(dataRows[i])
		b.WriteString("\n")
	}

	remaining := len(dataRows) - showCount
	if remaining > 0 {
		b.WriteString(fmt.Sprintf("... and %d more rows", remaining))
	}

	return b.String()
}

// --- Log detection and compression ---

func looksLikeLog(lines []string) bool {
	matchCount := 0
	checkCount := min(20, len(lines))
	for i := 0; i < checkCount; i++ {
		if logPattern.MatchString(lines[i]) || logLevelPattern.MatchString(lines[i]) {
			matchCount++
		}
	}
	return matchCount >= checkCount/2
}

func compressLog(lines []string) string {
	type logEntry struct {
		line  string
		count int
	}

	isImportant := func(line string) bool {
		upper := strings.ToUpper(line)
		return strings.Contains(upper, "ERROR") ||
			strings.Contains(upper, "WARN") ||
			strings.Contains(upper, "FATAL") ||
			strings.Contains(upper, "CRITICAL")
	}

	// Normalize line for dedup: strip timestamp prefix
	normalize := func(line string) string {
		loc := logPattern.FindStringIndex(line)
		if loc != nil {
			rest := strings.TrimSpace(line[loc[1]:])
			return rest
		}
		return line
	}

	stripDebug := len(lines) > 50
	var entries []logEntry
	seen := make(map[string]int) // normalized -> index in entries

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if stripDebug && !isImportant(line) {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE") {
				continue
			}
		}

		norm := normalize(line)
		if idx, ok := seen[norm]; ok {
			entries[idx].count++
		} else {
			seen[norm] = len(entries)
			entries = append(entries, logEntry{line: line, count: 1})
		}
	}

	// Truncate to last 30 unique lines
	if len(entries) > 30 {
		entries = entries[len(entries)-30:]
	}

	var b strings.Builder
	for _, e := range entries {
		if e.count > 1 {
			b.WriteString(fmt.Sprintf("%s (x%d)\n", e.line, e.count))
		} else {
			b.WriteString(e.line)
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// --- Markup detection and compression ---

func looksLikeMarkup(s string) bool {
	return strings.Contains(s, "</") || strings.Contains(s, "/>") || strings.Contains(s, "?>")
}

func compressMarkup(raw string, lines []string) string {
	if len(lines) <= 20 {
		return raw
	}
	kind := "XML"
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		kind = "HTML"
	}
	return fmt.Sprintf("%s response (%d bytes)", kind, len(raw))
}

// --- Plain text fallback ---

func compressPlainText(lines []string) string {
	total := len(lines)

	if total < 50 {
		return strings.Join(lines, "\n")
	}

	var headCount, tailCount int
	if total <= 200 {
		headCount = 20
		tailCount = 10
	} else {
		headCount = 15
		tailCount = 10
	}

	hidden := total - headCount - tailCount

	var b strings.Builder
	for i := 0; i < headCount; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("... (%d lines hidden)\n", hidden))
	for i := total - tailCount; i < total; i++ {
		b.WriteString(lines[i])
		if i < total-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

