package filters

import (
	"strings"
)

func filterDockerRmi(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	lines := strings.Split(trimmed, "\n")
	var out []string

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// Skip "Deleted: sha256:..." — internal layer IDs, no value to the user
		if strings.HasPrefix(t, "Deleted: sha256:") {
			continue
		}
		out = append(out, t)
	}

	if len(out) == 0 {
		return raw, nil
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
