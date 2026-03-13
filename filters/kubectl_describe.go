package filters

import (
	"fmt"
	"strings"
)

// describeSkipSections lists sections to completely strip from describe output.
var describeSkipSections = []string{
	"Annotations:",
	"Conditions:",
	"Tolerations:",
	"QoS Class:",
	"Node-Selectors:",
	"Priority:",
	"Service Account:",
	"IPs:",
}

// describeStripKeys lists sub-keys inside containers to strip.
var describeStripKeys = []string{
	"Limits:",
	"Requests:",
	"Liveness:",
	"Readiness:",
	"Startup:",
	"Environment:",
	"Mounts:",
	"Container ID:",
	"Image ID:",
	"Host Port:",
}

func filterKubectlDescribe(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	// Compress JSON output if kubectl describe was somehow invoked with json
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		compressed, err := compressJSON(trimmed)
		if err == nil {
			return compressed, nil
		}
		return raw, nil
	}

	if !looksLikeKubectlDescribeOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	var out []string
	lines := strings.Split(raw, "\n")

	skipSection := false   // in a top-level section to skip entirely
	inVolumes := false     // in Volumes section (keep names only)
	inEvents := false      // in Events section (filter warnings/normals)
	skipSubKey := false    // in a container sub-key to skip

	var eventWarnings []string
	var eventNormals []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect top-level keys (no leading whitespace)
		isTopLevel := len(line) > 0 && line[0] != ' ' && line[0] != '\t'

		// Reset flags on new top-level section
		if isTopLevel {
			skipSection = false
			inVolumes = false
			inEvents = false
			skipSubKey = false
		}

		// Check if this top-level section should be skipped
		if isTopLevel {
			for _, sec := range describeSkipSections {
				if strings.HasPrefix(line, sec) {
					skipSection = true
					break
				}
			}
		}

		if skipSection {
			continue
		}

		// Volumes section: keep volume names only
		if isTopLevel && strings.HasPrefix(line, "Volumes:") {
			inVolumes = true
			out = append(out, "Volumes:")
			continue
		}
		if inVolumes {
			if isTopLevel {
				inVolumes = false
			} else {
				indent := len(line) - len(strings.TrimLeft(line, " \t"))
				if indent <= 2 && strings.HasSuffix(trimmed, ":") {
					out = append(out, "  "+trimmed)
				}
				continue
			}
		}

		// Events section: collect and filter
		if isTopLevel && strings.HasPrefix(line, "Events:") {
			inEvents = true
			continue
		}
		if inEvents {
			if isTopLevel {
				inEvents = false
			} else {
				if strings.Contains(trimmed, "----") || (strings.Contains(trimmed, "Type") && strings.Contains(trimmed, "Reason")) {
					// Skip header/separator lines
				} else if strings.Contains(trimmed, "Warning") {
					eventWarnings = append(eventWarnings, trimmed)
				} else if strings.Contains(trimmed, "Normal") {
					eventNormals = append(eventNormals, trimmed)
				}
				continue
			}
		}

		// Strip verbose container sub-keys
		if !isTopLevel {
			for _, sk := range describeStripKeys {
				if strings.TrimSpace(line) == sk || strings.HasPrefix(strings.TrimSpace(line), sk+" ") {
					skipSubKey = true
					break
				}
			}
		}

		// When in a sub-key skip, detect end by finding next peer-level key
		if skipSubKey {
			// Sub-keys under Containers are indented ~4 spaces; their values ~6+
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if isTopLevel {
				skipSubKey = false
			} else if indent <= 4 && strings.Contains(trimmed, ":") {
				// New sub-key at container level - stop skipping
				skipSubKey = false
				// Check if this new sub-key should also be skipped
				for _, sk := range describeStripKeys {
					if trimmed == sk || strings.HasPrefix(trimmed, sk+" ") {
						skipSubKey = true
						break
					}
				}
				if skipSubKey {
					continue
				}
			} else {
				continue
			}
		}

		if !skipSection && !inVolumes && !inEvents {
			out = append(out, line)
		}
	}

	// Append filtered events
	if len(eventWarnings) > 0 || len(eventNormals) > 0 {
		out = append(out, "Events:")
		for _, w := range eventWarnings {
			out = append(out, "  "+w)
		}
		// Keep last 5 normal events
		start := 0
		if len(eventNormals) > 5 {
			start = len(eventNormals) - 5
			out = append(out, fmt.Sprintf("  (%d earlier Normal events hidden)", start))
		}
		for _, n := range eventNormals[start:] {
			out = append(out, "  "+n)
		}
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
