package filters

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var reGrepFileLine = regexp.MustCompile(`^(.+?):(\d+):(.*)$`)

const (
	maxFilesShown      = 10
	maxMatchesPerFile  = 2
)

func filterGrep(raw string) (string, error) {
	raw = stripAnsi(strings.TrimSpace(raw))
	if raw == "" {
		return "", nil
	}

	lines := strings.Split(raw, "\n")

	// Detect format: file:line:content (grep -rn) vs plain lines
	type fileMatch struct {
		file    string
		matches []string
	}

	fileOrder := []string{}
	fileMap := map[string]*fileMatch{}
	totalMatches := 0
	hasFilePrefix := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		m := reGrepFileLine.FindStringSubmatch(line)
		if m != nil {
			hasFilePrefix = true
			file := m[1]
			content := strings.TrimSpace(m[3])
			totalMatches++

			fm, ok := fileMap[file]
			if !ok {
				fm = &fileMatch{file: file}
				fileMap[file] = fm
				fileOrder = append(fileOrder, file)
			}
			fm.matches = append(fm.matches, content)
		} else {
			totalMatches++
		}
	}

	// If no file:line:content format detected, check if few enough to passthrough
	if !hasFilePrefix {
		if len(lines) <= 10 {
			return raw, nil
		}
		// Plain output - just truncate
		var out []string
		for i, line := range lines {
			if i >= 20 {
				out = append(out, fmt.Sprintf("... and %d more matches", len(lines)-20))
				break
			}
			out = append(out, line)
		}
		out = append(out, fmt.Sprintf("%d matches total", len(lines)))
		return strings.Join(out, "\n"), nil
	}

	totalFiles := len(fileOrder)

	// If few files and few matches, passthrough
	if totalFiles <= 2 && totalMatches <= 10 {
		return raw, nil
	}

	// Sort files by match count (most matches first)
	sort.Slice(fileOrder, func(i, j int) bool {
		return len(fileMap[fileOrder[i]].matches) > len(fileMap[fileOrder[j]].matches)
	})

	var out []string
	filesShown := 0
	for _, file := range fileOrder {
		if filesShown >= maxFilesShown {
			break
		}
		fm := fileMap[file]
		out = append(out, fmt.Sprintf("%s (%d matches)", fm.file, len(fm.matches)))
		for i, match := range fm.matches {
			if i >= maxMatchesPerFile {
				out = append(out, fmt.Sprintf("  ... +%d more", len(fm.matches)-maxMatchesPerFile))
				break
			}
			if len(match) > 80 {
				match = match[:80] + "..."
			}
			out = append(out, "  "+match)
		}
		filesShown++
	}

	if totalFiles > maxFilesShown {
		out = append(out, fmt.Sprintf("and %d more files", totalFiles-maxFilesShown))
	}

	out = append(out, fmt.Sprintf("%d matches across %d files", totalMatches, totalFiles))

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
