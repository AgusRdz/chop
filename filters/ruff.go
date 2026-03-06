package filters

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// ruff: src/app.py:1:1: F401 [*] `os` imported but unused
	// flake8: src/app.py:1:1: F401 'os' imported but unused
	reRuffProblem = regexp.MustCompile(`^(.+?):(\d+):\d+:\s+([A-Z]+\d+)\s+(.+)`)
	reRuffSummary = regexp.MustCompile(`(?i)^Found\s+(\d+)\s+errors?`)
	reRuffFixable = regexp.MustCompile(`(?i)\d+\s+fixable`)
)

func filterRuff(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "no problems", nil
	}
	if !looksLikeRuffOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	type ruffProblem struct {
		file string
		line string
		code string
		msg  string
	}

	var problems []ruffProblem
	var fixableMsg string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if reRuffFixable.MatchString(trimmed) {
			fixableMsg = trimmed
			continue
		}
		if reRuffSummary.MatchString(trimmed) {
			continue
		}

		if m := reRuffProblem.FindStringSubmatch(trimmed); m != nil {
			problems = append(problems, ruffProblem{
				file: m[1],
				line: m[2],
				code: m[3],
				msg:  m[4],
			})
		}
	}

	if len(problems) == 0 {
		return "no problems", nil
	}

	// Group by code
	codeMap := make(map[string][]string)
	codeMsg := make(map[string]string)
	for _, p := range problems {
		loc := fmt.Sprintf("%s:%s", p.file, p.line)
		codeMap[p.code] = append(codeMap[p.code], loc)
		if _, ok := codeMsg[p.code]; !ok {
			codeMsg[p.code] = p.msg
		}
	}

	codes := make([]string, 0, len(codeMap))
	for c := range codeMap {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	var out []string
	for _, code := range codes {
		locs := codeMap[code]
		out = append(out, fmt.Sprintf("%s (%d): %s", code, len(locs), strings.Join(locs, ", ")))
	}

	out = append(out, "")
	out = append(out, fmt.Sprintf("%d problems", len(problems)))
	if fixableMsg != "" {
		out = append(out, fixableMsg)
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

// filterPylint handles pylint-style output:
// src/app.py:1:0: C0114: Missing module docstring (missing-module-docstring)
var rePylintProblem = regexp.MustCompile(`^(.+?):(\d+):\d+:\s+([A-Z]\d+):\s+(.+?)\s+\((.+?)\)\s*$`)

func filterPylint(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "no problems", nil
	}
	if !looksLikePylintOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	type pylintProblem struct {
		file string
		line string
		code string
		name string
		msg  string
	}

	var problems []pylintProblem

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := rePylintProblem.FindStringSubmatch(trimmed); m != nil {
			problems = append(problems, pylintProblem{
				file: m[1],
				line: m[2],
				code: m[3],
				msg:  m[4],
				name: m[5],
			})
		}
	}

	if len(problems) == 0 {
		return raw, nil
	}

	// Group by code
	codeMap := make(map[string][]string)
	codeName := make(map[string]string)
	for _, p := range problems {
		loc := fmt.Sprintf("%s:%s", p.file, p.line)
		codeMap[p.code] = append(codeMap[p.code], loc)
		if _, ok := codeName[p.code]; !ok {
			codeName[p.code] = p.name
		}
	}

	codes := make([]string, 0, len(codeMap))
	for c := range codeMap {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	var out []string
	for _, code := range codes {
		locs := codeMap[code]
		out = append(out, fmt.Sprintf("%s/%s (%d): %s", code, codeName[code], len(locs), strings.Join(locs, ", ")))
	}

	out = append(out, "")
	out = append(out, fmt.Sprintf("%d problems", len(problems)))

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
