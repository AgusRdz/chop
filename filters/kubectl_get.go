package filters

import (
	"strings"
)

func filterKubectlGet(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "No resources found", nil
	}
	if !looksLikeKubectlGetOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	// Pass through JSON/YAML output but compress JSON if possible
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		compressed, err := compressJSON(raw)
		if err == nil {
			return compressed, nil
		}
		return raw, nil
	}
	if strings.HasPrefix(raw, "apiVersion:") || strings.HasPrefix(raw, "kind:") {
		return raw, nil
	}

	// Handle "No resources found" messages
	if strings.HasPrefix(raw, "No resources found") {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return raw, nil
	}

	header := lines[0]
	fields := strings.Fields(header)
	if len(fields) == 0 {
		return raw, nil
	}

	// Detect resource type from header and pick columns to keep
	keepCols := detectColumnsToKeep(fields)

	// Find column boundaries from header using position-based parsing
	colBounds := findColumnBoundaries(header, fields)

	var out []string
	// Build filtered header
	var headerParts []string
	for _, idx := range keepCols {
		if idx < len(fields) {
			headerParts = append(headerParts, fields[idx])
		}
	}
	out = append(out, strings.Join(headerParts, "\t"))

	// Process data lines
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var parts []string
		for _, colIdx := range keepCols {
			if colIdx < len(colBounds) {
				start := colBounds[colIdx].start
				end := colBounds[colIdx].end
				val := safeExtract(line, start, end)
				parts = append(parts, val)
			}
		}
		out = append(out, strings.Join(parts, "\t"))
	}

	if len(out) <= 1 {
		return "No resources found", nil
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

type colBound struct {
	start int
	end   int
}

func findColumnBoundaries(header string, fields []string) []colBound {
	bounds := make([]colBound, len(fields))
	searchFrom := 0
	for i, f := range fields {
		idx := strings.Index(header[searchFrom:], f)
		if idx == -1 {
			bounds[i] = colBound{start: len(header), end: len(header)}
			continue
		}
		start := searchFrom + idx
		bounds[i] = colBound{start: start}
		searchFrom = start + len(f)
	}
	// Set end boundaries
	for i := 0; i < len(bounds)-1; i++ {
		bounds[i].end = bounds[i+1].start
	}
	if len(bounds) > 0 {
		bounds[len(bounds)-1].end = -1 // -1 means end of line
	}
	return bounds
}

func safeExtract(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end == -1 || end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

func detectColumnsToKeep(fields []string) []int {
	headerStr := strings.Join(fields, " ")

	// Pods: NAME STATUS RESTARTS AGE
	if containsAll(headerStr, "READY", "STATUS", "RESTARTS") {
		return findIndices(fields, "NAME", "STATUS", "RESTARTS", "AGE")
	}

	// Deployments: NAME READY UP-TO-DATE AGE
	if containsAll(headerStr, "UP-TO-DATE", "AVAILABLE") {
		return findIndices(fields, "NAME", "READY", "UP-TO-DATE", "AGE")
	}

	// Services: NAME TYPE CLUSTER-IP PORT(S)
	if containsAll(headerStr, "TYPE", "CLUSTER-IP", "PORT(S)") {
		return findIndices(fields, "NAME", "TYPE", "CLUSTER-IP", "PORT(S)")
	}

	// Nodes: NAME STATUS ROLES AGE
	if containsAll(headerStr, "ROLES", "VERSION") {
		return findIndices(fields, "NAME", "STATUS", "ROLES", "AGE")
	}

	// Generic: keep first 4 columns max
	max := 4
	if len(fields) < max {
		max = len(fields)
	}
	indices := make([]int, max)
	for i := range indices {
		indices[i] = i
	}
	return indices
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func findIndices(fields []string, names ...string) []int {
	fieldMap := make(map[string]int, len(fields))
	for i, f := range fields {
		fieldMap[f] = i
	}
	indices := make([]int, 0, len(names))
	for _, name := range names {
		if i, ok := fieldMap[name]; ok {
			indices = append(indices, i)
		}
	}
	return indices
}
