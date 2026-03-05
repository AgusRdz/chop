package filters

import (
	"fmt"
	"strings"
)

func getGcloudFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterGcloudGeneric
	}
	switch args[0] {
	case "compute":
		if len(args) > 1 && args[1] == "instances" && len(args) > 2 && args[2] == "list" {
			return filterGcloudInstancesList
		}
		return filterGcloudGeneric
	default:
		return filterGcloudGeneric
	}
}

// filterGcloudGeneric auto-detects table vs JSON and compresses accordingly.
func filterGcloudGeneric(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isGcloudError(raw) {
		return raw, nil
	}

	// Try JSON first
	if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		compressed, err := compressJSON(raw)
		if err == nil {
			return compressed, nil
		}
	}

	// Table output — truncate rows
	return filterGcloudTable(raw), nil
}

// filterGcloudInstancesList extracts NAME ZONE STATUS from table or JSON.
func filterGcloudInstancesList(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isGcloudError(raw) {
		return raw, nil
	}

	// Try JSON format
	if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		compressed, err := compressJSON(raw)
		if err == nil {
			return compressed, nil
		}
	}

	// Table format: keep header + data rows, extract NAME ZONE STATUS columns
	lines := strings.Split(raw, "\n")
	var nonEmpty []string
	for _, l := range lines {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}

	if len(nonEmpty) == 0 {
		return raw, nil
	}

	// First line is header
	header := nonEmpty[0]
	dataLines := nonEmpty[1:]

	// Find column positions for NAME, ZONE, STATUS
	nameIdx, zoneIdx, statusIdx := findColumnIndices(header)

	if nameIdx < 0 {
		// Can't parse columns, fall back to table truncation
		return filterGcloudTable(raw), nil
	}

	var out []string
	out = append(out, formatInstanceHeader())

	for _, line := range dataLines {
		fields := strings.Fields(line)
		name := fieldAt(fields, nameIdx)
		zone := fieldAt(fields, zoneIdx)
		status := fieldAt(fields, statusIdx)
		out = append(out, fmt.Sprintf("%s  %s  %s", name, zone, status))
	}

	return fmt.Sprintf("Instances (%d):\n%s", len(dataLines), strings.Join(out, "\n")), nil
}

func formatInstanceHeader() string {
	return "NAME  ZONE  STATUS"
}

func findColumnIndices(header string) (nameIdx, zoneIdx, statusIdx int) {
	nameIdx = -1
	zoneIdx = -1
	statusIdx = -1

	fields := strings.Fields(header)
	for i, f := range fields {
		switch f {
		case "NAME":
			nameIdx = i
		case "ZONE":
			zoneIdx = i
		case "STATUS":
			statusIdx = i
		}
	}
	return
}

func fieldAt(fields []string, idx int) string {
	if idx >= 0 && idx < len(fields) {
		return fields[idx]
	}
	return ""
}

// filterGcloudTable keeps header + first 10 rows + count for table output.
func filterGcloudTable(raw string) string {
	lines := strings.Split(raw, "\n")
	var nonEmpty []string
	for _, l := range lines {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}

	if len(nonEmpty) <= 1 {
		return raw
	}

	// First line is header, rest are data
	header := nonEmpty[0]
	dataLines := nonEmpty[1:]

	// Check for separator line (like "---")
	startIdx := 0
	extraHeader := ""
	if len(dataLines) > 0 && isSeparatorLine(dataLines[0]) {
		extraHeader = "\n" + dataLines[0]
		dataLines = dataLines[1:]
	}

	maxRows := 10
	if len(dataLines) <= maxRows {
		return strings.Join(nonEmpty, "\n")
	}

	var out []string
	out = append(out, header+extraHeader)
	for i := startIdx; i < maxRows && i < len(dataLines); i++ {
		out = append(out, dataLines[i])
	}
	out = append(out, fmt.Sprintf("... (%d more rows, %d total)", len(dataLines)-maxRows, len(dataLines)))

	return strings.Join(out, "\n")
}

func isGcloudError(raw string) bool {
	return strings.Contains(raw, "ERROR:") ||
		strings.Contains(raw, "error:") ||
		strings.HasPrefix(raw, "ERROR") ||
		strings.Contains(raw, "PERMISSION_DENIED") ||
		strings.Contains(raw, "NOT_FOUND") ||
		strings.Contains(raw, "could not be found")
}
