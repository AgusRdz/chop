package filters

import (
	"fmt"
	"strings"
)

func filterDockerPs(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "no running containers", nil
	}
	if !looksLikeDockerPsOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return "no running containers", nil
	}

	header := lines[0]

	// Find column positions dynamically from the header
	nameIdx := strings.Index(header, "NAMES")
	imageIdx := strings.Index(header, "IMAGE")
	statusIdx := strings.Index(header, "STATUS")

	if nameIdx == -1 || imageIdx == -1 || statusIdx == -1 {
		// If we can't parse the header, return raw
		return raw, nil
	}

	// Standard docker ps has IMAGE before STATUS before NAMES.
	// Custom --format can reorder columns, breaking the bounds math - return raw.
	if imageIdx > statusIdx || statusIdx > nameIdx {
		return raw, nil
	}

	// Find the end of each column by looking at the next column start
	// Columns in docker ps: CONTAINER ID, IMAGE, COMMAND, CREATED, STATUS, PORTS, NAMES
	// We need IMAGE, STATUS, NAMES

	dataLines := lines[1:]
	if len(dataLines) == 0 {
		return "no running containers", nil
	}

	// Find column boundaries
	commandIdx := strings.Index(header, "COMMAND")
	portsIdx := strings.Index(header, "PORTS")

	imageBound := statusIdx
	if commandIdx > imageIdx {
		imageBound = commandIdx
	}

	statusBound := portsIdx
	if statusBound == -1 {
		statusBound = nameIdx
	}

	var out []string
	for _, line := range dataLines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// NAMES is last column, extend to end of line (not header)
		name := extractColumn(line, nameIdx, len(line))
		image := extractColumn(line, imageIdx, imageBound)
		status := extractColumn(line, statusIdx, statusBound)

		out = append(out, fmt.Sprintf("%s (%s) %s", name, image, status))
	}

	if len(out) == 0 {
		return "no running containers", nil
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

// extractColumn extracts text from a fixed-width table line between start and end positions.
func extractColumn(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

// findNextColAfter finds the start of the next column header after the given position.
func findNextColAfter(header string, afterIdx int) int {
	// Known docker ps columns in order
	cols := []string{"CONTAINER ID", "IMAGE", "COMMAND", "CREATED", "STATUS", "PORTS", "NAMES"}
	for _, col := range cols {
		idx := strings.Index(header, col)
		if idx > afterIdx {
			return idx
		}
	}
	return len(header)
}
