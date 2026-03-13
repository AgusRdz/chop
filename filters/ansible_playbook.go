package filters

import (
	"strings"
)

// filterAnsiblePlaybook compresses ansible-playbook output by removing "ok: [host]"
// and "skipping: [host]" messages, and keeping failures, changes, and headers.
func filterAnsiblePlaybook(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	lines := strings.Split(trimmed, "\n")
	var out []string

	var currentHeader string
	headerPrinted := false
	inRecap := false

	// Ansible task output often starts with PLAY or TASK followed by task details.
	for _, line := range lines {
		l := strings.TrimRight(line, "\r")
		trimmedLine := strings.TrimSpace(l)

		// Check if we are entering the play recap
		if strings.HasPrefix(l, "PLAY RECAP *") {
			inRecap = true
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			out = append(out, l)
			continue
		}

		// Inside PLAY RECAP - unconditionally keep everything
		if inRecap {
			out = append(out, l)
			continue
		}

		// Always keep play and task headers, but remember them to only print if
		// followed by relevant non-ok output.
		if strings.HasPrefix(l, "PLAY [") || strings.HasPrefix(l, "TASK [") || strings.HasPrefix(l, "RUNNING HANDLER [") {
			currentHeader = l
			headerPrinted = false
			continue
		}

		// Skip successful "ok" and "skipping" lines if we're not in recap
		if strings.HasPrefix(trimmedLine, "ok: [") || strings.HasPrefix(trimmedLine, "skipping: [") {
			continue
		}

		// For anything else (changed, failed, fatal, arbitrary stdout/stderr)
		if trimmedLine != "" {
			if !headerPrinted && currentHeader != "" {
				// Don't inject multiple blank lines
				if len(out) > 0 && out[len(out)-1] != "" {
					out = append(out, "")
				}
				out = append(out, currentHeader)
				headerPrinted = true
			}
			out = append(out, l)
		}
	}

	result := strings.Join(out, "\n")
	if result == "" {
		// Fallback if everything was 'ok'
		return "Playbook finished successfully (all tasks ok/skipped).", nil
	}

	return outputSanityCheck(raw, result), nil
}
