package filters

import (
	"fmt"
	"strings"
)

// No regex needed - using strings operations

func filterKubectlApply(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeKubectlApplyOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	groups := make(map[string][]string) // action -> []resource
	var warnings []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmed), "warning:") {
			warnings = append(warnings, trimmed)
			continue
		}

		// Match lines like "resource/name created" or "resource/name configured"
		for _, action := range []string{"created", "configured", "unchanged", "replaced", "deleted"} {
			if strings.HasSuffix(trimmed, " "+action) {
				resource := strings.TrimSuffix(trimmed, " "+action)
				groups[action] = append(groups[action], resource)
				break
			}
		}
	}

	total := 0
	for _, v := range groups {
		total += len(v)
	}
	if total == 0 {
		return raw, nil
	}

	var out []string
	for _, action := range []string{"created", "configured", "replaced", "unchanged", "deleted"} {
		resources := groups[action]
		if len(resources) == 0 {
			continue
		}
		out = append(out, fmt.Sprintf("%s(%d): %s", action, len(resources), strings.Join(resources, ", ")))
	}

	for _, w := range warnings {
		out = append(out, w)
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func filterKubectlDelete(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeKubectlDeleteOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var resources []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, " deleted") {
			resource := strings.TrimSuffix(trimmed, " deleted")
			// Clean up quoted names: pod "my-pod" -> pod/my-pod
			resource = strings.ReplaceAll(resource, " \"", "/")
			resource = strings.ReplaceAll(resource, "\"", "")
			resources = append(resources, resource)
		}
	}

	if len(resources) == 0 {
		return raw, nil
	}

	result := fmt.Sprintf("deleted %d resources: %s", len(resources), strings.Join(resources, ", "))
	return outputSanityCheck(raw, result), nil
}
