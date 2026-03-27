package filters

import (
	"fmt"
	"regexp"
	"strings"
)

// logSecondsPattern strips the seconds portion (:SS or :SS.mmm) left behind
// by logPattern which only captures HH:MM.
var logSecondsPattern = regexp.MustCompile(`^:\d{2}(\.\d+)?\s*`)

// Pattern regexes compiled once at package level.
// Order matters: more specific patterns must come before generic ones.
var patternRegexes = []*regexp.Regexp{
	// UUIDs: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),
	// IP:port
	regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?\b`),
	// Durations: 45ms, 1.2s, 300us, etc (before generic numbers)
	regexp.MustCompile(`\b\d+(\.\d+)?(ms|ns|us|µs|s|m|h)\b`),
	// ISO timestamps that survived initial strip: 2024-03-11T10:00:01.123Z
	regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`),
	// Date-like: 2024-03-11, 2024/03/11
	regexp.MustCompile(`\b\d{4}[-/]\d{2}[-/]\d{2}\b`),
	// Hex IDs (8+ chars, lowercase or mixed)
	regexp.MustCompile(`\b[0-9a-fA-F]{8,}\b`),
	// Key=value pairs (preserve key, replace value)
	regexp.MustCompile(`(\w+=)\S+`),
	// Quoted strings
	regexp.MustCompile(`"[^"]*"`),
	regexp.MustCompile(`'[^']*'`),
	// Pure numbers (integers and decimals)
	regexp.MustCompile(`\b\d+(\.\d+)?\b`),
}

// keyValuePatternIdx is the index of the key=value regex in patternRegexes,
// used to avoid per-line string comparison in patternFingerprint.
const keyValuePatternIdx = 6

const placeholder = "<*>"

// patternFingerprint replaces variable parts of a log line with <*> placeholders,
// producing a structural "fingerprint" for grouping similar lines.
func patternFingerprint(line string) string {
	result := line
	for i, re := range patternRegexes {
		if i == keyValuePatternIdx {
			// Special handling: preserve the key= part
			result = re.ReplaceAllString(result, "${1}"+placeholder)
		} else {
			result = re.ReplaceAllString(result, placeholder)
		}
	}
	return result
}

// hasEnoughStaticTokens checks that a fingerprint retains enough non-placeholder
// tokens to be a meaningful pattern (prevents over-grouping).
func hasEnoughStaticTokens(fingerprint string) bool {
	tokens := strings.Fields(fingerprint)
	if len(tokens) == 0 {
		return false
	}
	static := 0
	for _, t := range tokens {
		if t != placeholder && !strings.Contains(t, placeholder) {
			static++
		}
	}
	return static >= 2
}

type patternGroup struct {
	fingerprint string
	lastLine    string // most recent representative line
	count       int
	isError     bool
}

// compressLogPatterns applies pattern-based grouping to log lines.
// It returns the compressed output and true if pattern compression was applied,
// or empty string and false if the input doesn't benefit from pattern grouping.
func compressLogPatterns(lines []string, isImportantFn func(string) bool) (string, bool) {
	if len(lines) < 10 {
		return "", false
	}

	type lineInfo struct {
		original    string
		normalized  string
		fingerprint string
		isImportant bool
	}

	infos := make([]lineInfo, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Strip timestamp for normalization (reuse existing logPattern)
		norm := trimmed
		loc := logPattern.FindStringIndex(trimmed)
		if loc != nil {
			norm = strings.TrimSpace(trimmed[loc[1]:])
			// logPattern captures HH:MM but not :SS - strip leftover seconds
			norm = logSecondsPattern.ReplaceAllString(norm, "")
		}

		fp := patternFingerprint(norm)
		infos = append(infos, lineInfo{
			original:    trimmed,
			normalized:  norm,
			fingerprint: fp,
			isImportant: isImportantFn(trimmed),
		})
	}

	if len(infos) == 0 {
		return "", false
	}

	// Count unique fingerprints to decide if pattern grouping helps
	fpCounts := make(map[string]int)
	for _, info := range infos {
		if hasEnoughStaticTokens(info.fingerprint) {
			fpCounts[info.fingerprint]++
		}
	}

	// Check if pattern grouping provides meaningful compression:
	// at least one pattern must appear 3+ times
	hasRepeats := false
	for _, count := range fpCounts {
		if count >= 3 {
			hasRepeats = true
			break
		}
	}
	if !hasRepeats {
		return "", false
	}

	// Build pattern groups preserving order of first appearance
	groupOrder := []string{}
	groups := make(map[string]*patternGroup)

	for _, info := range infos {
		fp := info.fingerprint
		if !hasEnoughStaticTokens(fp) {
			// Treat as unique - use the original line as fingerprint
			fp = info.original
		}

		if g, ok := groups[fp]; ok {
			g.count++
			g.lastLine = info.original
			if info.isImportant {
				g.isError = true
			}
		} else {
			groups[fp] = &patternGroup{
				fingerprint: fp,
				lastLine:    info.original,
				count:       1,
				isError:     info.isImportant,
			}
			groupOrder = append(groupOrder, fp)
		}
	}

	// Separate error groups and normal groups
	var errorGroups []*patternGroup
	var normalGroups []*patternGroup
	for _, fp := range groupOrder {
		g := groups[fp]
		if g.isError {
			errorGroups = append(errorGroups, g)
		} else {
			normalGroups = append(normalGroups, g)
		}
	}

	// Truncate normal groups to last 30
	hidden := 0
	if len(normalGroups) > 30 {
		hidden = len(normalGroups) - 30
		normalGroups = normalGroups[hidden:]
	}

	var b strings.Builder
	if hidden > 0 {
		b.WriteString(fmt.Sprintf("(%d earlier patterns hidden)\n", hidden))
	}

	writeGroup := func(g *patternGroup) {
		if g.count == 1 {
			// Single occurrence - show original line unchanged
			b.WriteString(g.lastLine)
		} else {
			// Multiple occurrences - show a real example with count
			b.WriteString(fmt.Sprintf("%s (x%d)", g.lastLine, g.count))
		}
		b.WriteString("\n")
	}

	// Errors first
	for _, g := range errorGroups {
		writeGroup(g)
	}
	for _, g := range normalGroups {
		writeGroup(g)
	}

	return strings.TrimRight(b.String(), "\n"), true
}
