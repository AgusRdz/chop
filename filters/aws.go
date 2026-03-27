package filters

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Pre-compiled regexes for S3 ls parsing (avoid recompilation per call).
var (
	reS3File = regexp.MustCompile(`^\s*\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+(\d+)\s+(.+)$`)
	reS3Pre  = regexp.MustCompile(`^\s*PRE\s+(.+)$`)
)

func getAwsFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterAwsGeneric
	}
	switch args[0] {
	case "s3":
		if len(args) > 1 && args[1] == "ls" {
			return filterAwsS3Ls
		}
		return filterAwsGeneric
	case "ec2":
		if len(args) > 1 && args[1] == "describe-instances" {
			return filterAwsEc2Describe
		}
		return filterAwsGeneric
	case "logs":
		return filterAwsLogs
	default:
		return filterAwsGeneric
	}
}

// filterAwsGeneric compresses any AWS JSON output; preserves errors.
func filterAwsGeneric(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAwsError(raw) {
		return raw, nil
	}
	compressed, err := compressJSON(raw)
	if err != nil {
		// Not JSON - return as-is
		return raw, nil
	}
	return compressed, nil
}

// filterAwsS3Ls groups s3 ls output by prefix and shows count + total size.
func filterAwsS3Ls(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAwsError(raw) {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")

	// Detect prefix-only lines (PRE prefix/) vs file lines
	type prefixStats struct {
		count     int
		totalSize int64
	}
	prefixes := make(map[string]*prefixStats)
	var prefixOrder []string
	var otherLines []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if m := reS3Pre.FindStringSubmatch(line); m != nil {
			otherLines = append(otherLines, "PRE "+m[1])
			continue
		}
		if m := reS3File.FindStringSubmatch(line); m != nil {
			var size int64
			fmt.Sscanf(m[1], "%d", &size)
			filePath := m[2]
			// Group by first path component (prefix)
			prefix := "."
			if idx := strings.Index(filePath, "/"); idx >= 0 {
				prefix = filePath[:idx]
			}
			if _, ok := prefixes[prefix]; !ok {
				prefixes[prefix] = &prefixStats{}
				prefixOrder = append(prefixOrder, prefix)
			}
			prefixes[prefix].count++
			prefixes[prefix].totalSize += size
		} else {
			otherLines = append(otherLines, line)
		}
	}

	// If no grouping happened, just return truncated lines
	if len(prefixes) == 0 {
		if len(otherLines) > 20 {
			result := strings.Join(otherLines[:20], "\n")
			result += fmt.Sprintf("\n... (%d more items)", len(otherLines)-20)
			return result, nil
		}
		return strings.Join(otherLines, "\n"), nil
	}

	var out []string
	if len(otherLines) > 0 {
		out = append(out, otherLines...)
	}
	sort.Strings(prefixOrder)
	for _, prefix := range prefixOrder {
		s := prefixes[prefix]
		out = append(out, fmt.Sprintf("%s/ - %d files, %s", prefix, s.count, humanSize(s.totalSize)))
	}
	// Files at root level (prefix ".")
	if s, ok := prefixes["."]; ok {
		// Already included above, but rename label
		out[len(out)-1] = fmt.Sprintf("(root) - %d files, %s", s.count, humanSize(s.totalSize))
	}

	return strings.Join(out, "\n"), nil
}

func humanSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// filterAwsEc2Describe extracts ID STATE TYPE NAME per instance.
func filterAwsEc2Describe(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAwsError(raw) {
		return raw, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return raw, nil
	}

	reservations, ok := data["Reservations"].([]interface{})
	if !ok {
		// Fallback to generic compression
		return compressJSON(raw)
	}

	var lines []string
	for _, r := range reservations {
		res, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		instances, ok := res["Instances"].([]interface{})
		if !ok {
			continue
		}
		for _, inst := range instances {
			i, ok := inst.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := i["InstanceId"].(string)
			itype, _ := i["InstanceType"].(string)
			state := ""
			if s, ok := i["State"].(map[string]interface{}); ok {
				state, _ = s["Name"].(string)
			}
			name := extractNameTag(i)
			lines = append(lines, fmt.Sprintf("%s  %s  %s  %s", id, state, itype, name))
		}
	}

	if len(lines) == 0 {
		return "No instances found", nil
	}
	header := fmt.Sprintf("EC2 Instances (%d):", len(lines))
	result := header + "\n" + strings.Join(lines, "\n")
	return outputSanityCheck(raw, result), nil
}

func extractNameTag(instance map[string]interface{}) string {
	tags, ok := instance["Tags"].([]interface{})
	if !ok {
		return "(no name)"
	}
	for _, t := range tags {
		tag, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		if key, _ := tag["Key"].(string); key == "Name" {
			val, _ := tag["Value"].(string)
			return val
		}
	}
	return "(no name)"
}

// filterAwsLogs strips verbose timestamps and deduplicates repeated messages.
func filterAwsLogs(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAwsError(raw) {
		return raw, nil
	}

	// Try JSON (e.g., aws logs get-log-events returns JSON)
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		if events, ok := data["events"].([]interface{}); ok {
			return filterLogEvents(events), nil
		}
		return compressJSON(raw)
	}

	// Text output - deduplicate lines
	return deduplicateLines(raw), nil
}

func filterLogEvents(events []interface{}) string {
	type msgCount struct {
		msg   string
		count int
	}
	seen := make(map[string]*msgCount)
	var order []string

	for _, e := range events {
		ev, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		msg, _ := ev["message"].(string)
		msg = strings.TrimSpace(msg)
		// Strip leading timestamp patterns like "2024-01-15T10:30:00.000Z "
		if idx := strings.Index(msg, " "); idx > 0 && idx < 30 {
			candidate := msg[:idx]
			if len(candidate) > 10 && (strings.Contains(candidate, "T") || strings.Contains(candidate, ":")) {
				msg = msg[idx+1:]
			}
		}
		if msg == "" {
			continue
		}
		if _, ok := seen[msg]; !ok {
			seen[msg] = &msgCount{msg: msg, count: 0}
			order = append(order, msg)
		}
		seen[msg].count++
	}

	var lines []string
	for _, key := range order {
		mc := seen[key]
		if mc.count > 1 {
			lines = append(lines, fmt.Sprintf("[x%d] %s", mc.count, mc.msg))
		} else {
			lines = append(lines, mc.msg)
		}
	}

	header := fmt.Sprintf("Log events (%d, %d unique):", len(events), len(order))
	return header + "\n" + strings.Join(lines, "\n")
}

func deduplicateLines(raw string) string {
	lines := strings.Split(raw, "\n")
	type lineCount struct {
		line  string
		count int
	}
	seen := make(map[string]*lineCount)
	var order []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; !ok {
			seen[trimmed] = &lineCount{line: trimmed, count: 0}
			order = append(order, trimmed)
		}
		seen[trimmed].count++
	}

	var out []string
	for _, key := range order {
		lc := seen[key]
		if lc.count > 1 {
			out = append(out, fmt.Sprintf("[x%d] %s", lc.count, lc.line))
		} else {
			out = append(out, lc.line)
		}
	}
	return strings.Join(out, "\n")
}

func isAwsError(raw string) bool {
	return strings.Contains(raw, "An error occurred") ||
		strings.Contains(raw, "AccessDenied") ||
		strings.Contains(raw, "NoSuchBucket") ||
		strings.Contains(raw, "NoSuchKey") ||
		strings.Contains(raw, "InvalidParameterValue") ||
		strings.Contains(raw, "UnauthorizedAccess") ||
		(strings.Contains(raw, "\"Error\"") && strings.Contains(raw, "\"Code\""))
}
