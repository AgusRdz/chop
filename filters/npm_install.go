package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reAdded       = regexp.MustCompile(`added (\d+) packages?`)
	reRemoved     = regexp.MustCompile(`removed (\d+) packages?`)
	reChanged     = regexp.MustCompile(`changed (\d+) packages?`)
	reUpdated     = regexp.MustCompile(`updated (\d+) packages?`)
	reAuditClean  = regexp.MustCompile(`found 0 vulnerabilities`)
	reAuditIssues = regexp.MustCompile(`(\d+ vulnerabilities? \(.*?\))`)
	reNpmWarn     = regexp.MustCompile(`(?i)^npm warn`)
	reNpmErr      = regexp.MustCompile(`(?i)^npm error|^ERR!`)
)

func filterNpmInstall(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}
	if !looksLikeNpmInstallOutput(trimmed) {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")

	var warnings []string
	var errors []string
	var summaryLine string
	var auditLine string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Capture warnings
		if reNpmWarn.MatchString(trimmed) {
			warnings = append(warnings, trimmed)
			continue
		}

		// Capture errors
		if reNpmErr.MatchString(trimmed) {
			errors = append(errors, trimmed)
			continue
		}

		// Capture summary line (e.g., "added 150 packages in 12s")
		if reAdded.MatchString(trimmed) || reRemoved.MatchString(trimmed) || reChanged.MatchString(trimmed) || reUpdated.MatchString(trimmed) {
			summaryLine = trimmed
			continue
		}

		// Capture audit info
		if reAuditClean.MatchString(trimmed) {
			auditLine = ""
			continue
		}
		if reAuditIssues.MatchString(trimmed) {
			auditLine = trimmed
			continue
		}
		// Alternative audit format: "X vulnerabilities (Y low, Z moderate)"
		if strings.Contains(trimmed, "vulnerabilities") && !strings.Contains(trimmed, "found 0") {
			auditLine = trimmed
			continue
		}
	}

	var out strings.Builder

	// Build compact summary
	if summaryLine != "" {
		// Extract counts from summary
		var parts []string
		if m := reAdded.FindStringSubmatch(summaryLine); m != nil {
			parts = append(parts, fmt.Sprintf("added %s packages", m[1]))
		}
		if m := reRemoved.FindStringSubmatch(summaryLine); m != nil {
			parts = append(parts, fmt.Sprintf("removed %s packages", m[1]))
		}
		if m := reChanged.FindStringSubmatch(summaryLine); m != nil {
			parts = append(parts, fmt.Sprintf("changed %s packages", m[1]))
		}
		if m := reUpdated.FindStringSubmatch(summaryLine); m != nil {
			parts = append(parts, fmt.Sprintf("updated %s packages", m[1]))
		}

		summary := strings.Join(parts, ", ")
		if auditLine != "" {
			fmt.Fprintf(&out, "%s (%s)", summary, auditLine)
		} else {
			out.WriteString(summary)
		}
	} else if len(errors) == 0 && len(warnings) == 0 {
		// No summary found and no issues - fallback
		return raw, nil
	}

	// Append errors (keep full text - these are actionable)
	for _, e := range errors {
		fmt.Fprintf(&out, "\n%s", e)
	}

	// Condense warnings to a count
	if len(warnings) > 0 {
		fmt.Fprintf(&out, "\n%d warnings", len(warnings))
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}
