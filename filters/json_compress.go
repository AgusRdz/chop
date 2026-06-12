package filters

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// truncateRunes truncates s to at most n runes, appending "..." if truncated.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "..."
}

// compressJSON takes a JSON string and returns a structural summary with types.
// Small/simple JSON is returned as-is. Large JSON gets structure-only compression.
func compressJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Redact sensitive data before compression/summarization
	parsed = redactJSON(parsed)

	// For small JSON (< 5 keys at top level, no large arrays), preserve values
	if isSmallJSON(parsed) {
		// Re-marshal with indent for readability but keep it compact
		b, err := json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return raw, nil
		}
		return string(b), nil
	}

	return compressJSONValue(parsed, 0), nil
}

// isSmallJSON returns true if the JSON is small enough to preserve values.
func isSmallJSON(v interface{}) bool {
	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) > 5 {
			return false
		}
		for _, child := range val {
			if arr, ok := child.([]interface{}); ok && len(arr) > 3 {
				return false
			}
			if obj, ok := child.(map[string]interface{}); ok && len(obj) > 5 {
				return false
			}
		}
		return true
	case []interface{}:
		return len(val) <= 3
	default:
		return true
	}
}

// compressJSONValue recursively summarizes a JSON value.
// depth controls how deep we show structure (max 2 levels for nested objects).
func compressJSONValue(v interface{}, depth int) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return fmt.Sprintf("%v", val)
	case float64:
		// Show integer form if no decimal part
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case string:
		if utf8.RuneCountInString(val) > 50 {
			return fmt.Sprintf("%q", truncateRunes(val, 50))
		}
		return fmt.Sprintf("%q", val)
	case []interface{}:
		return compressArray(val, depth)
	case map[string]interface{}:
		return compressObject(val, depth)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func compressArray(arr []interface{}, depth int) string {
	if len(arr) == 0 {
		return "[]"
	}

	// Show first element structure + count
	elemSummary := compressJSONValue(arr[0], depth+1)
	if len(arr) == 1 {
		return fmt.Sprintf("[%s]", elemSummary)
	}
	return fmt.Sprintf("[%s x%d]", elemSummary, len(arr))
}

func compressObject(obj map[string]interface{}, depth int) string {
	if len(obj) == 0 {
		return "{}"
	}

	// Beyond depth 2, just summarize as {object(N keys)}
	if depth > 2 {
		return fmt.Sprintf("{object(%d keys)}", len(obj))
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		valSummary := compressValueType(obj[k], depth+1)
		parts = append(parts, fmt.Sprintf("%q: %s", k, valSummary))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// compressValueType returns a type summary for object values.
// When typeOnly is true, scalar values are replaced with type names.
func compressValueType(v interface{}, depth int) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		elemType := describeType(val[0], depth)
		return fmt.Sprintf("[%s x%d]", elemType, len(val))
	case map[string]interface{}:
		if depth > 2 {
			return fmt.Sprintf("{object(%d keys)}", len(val))
		}
		return compressObject(val, depth)
	default:
		return "unknown"
	}
}

// describeType gives a brief type description for array element summaries.
func describeType(v interface{}, depth int) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return fmt.Sprintf("array(%d)", len(val))
	case map[string]interface{}:
		if depth > 2 {
			return fmt.Sprintf("object(%d keys)", len(val))
		}
		return compressObject(val, depth)
	default:
		return "unknown"
	}
}
