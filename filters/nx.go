package filters

import (
	"regexp"
	"strings"
)

var (
	reNxSuccess = regexp.MustCompile(`(?i)NX\s+Successfully ran target\s+(\w+)\s+for project\s+(\S+)\s+\((.+?)\)`)
	reNxFailed  = regexp.MustCompile(`(?i)NX\s+Ran target\s+(\w+)\s+for project\s+(\S+)\s+\((.+?)\)`)
)

func filterNxBuild(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeNxBuildOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var resultLine string
	var errors []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := reNxSuccess.FindStringSubmatch(trimmed); m != nil {
			resultLine = "nx " + m[1] + " " + m[2] + " ok (" + m[3] + ")"
		}
		if m := reNxFailed.FindStringSubmatch(trimmed); m != nil {
			if resultLine == "" {
				resultLine = "nx " + m[1] + " " + m[2] + " (" + m[3] + ")"
			}
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "error") {
			errors = append(errors, trimmed)
		}
	}

	if resultLine == "" {
		return raw, nil
	}

	var out []string
	for _, e := range errors {
		out = append(out, e)
	}
	out = append(out, resultLine)

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func filterNxTest(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeNxTestOutput(trimmed) {
		return raw, nil
	}

	// Nx test wraps jest output - delegate to npm test filter
	// but also extract the nx result line
	raw = trimmed

	// Try to get jest summary from the inner output
	jestResult, _ := filterNpmTestCmd(raw)

	// Also get the nx wrapper line
	var nxLine string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := reNxSuccess.FindStringSubmatch(trimmed); m != nil {
			nxLine = "nx " + m[1] + " " + m[2] + " ok (" + m[3] + ")"
		}
		if m := reNxFailed.FindStringSubmatch(trimmed); m != nil {
			if nxLine == "" {
				nxLine = "nx " + m[1] + " " + m[2] + " (" + m[3] + ")"
			}
		}
	}

	// If jest filter compressed it, use that + nx line
	if jestResult != raw && jestResult != "" {
		if nxLine != "" {
			jestResult += "\n" + nxLine
		}
		return outputSanityCheck(raw, jestResult), nil
	}

	if nxLine != "" {
		return outputSanityCheck(raw, nxLine), nil
	}

	return raw, nil
}
