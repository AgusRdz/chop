package filters

import (
	"encoding/json"
	"fmt"
	"strings"
)

func getAzFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterAzGeneric
	}
	switch args[0] {
	case "vm":
		if len(args) > 1 && args[1] == "list" {
			return filterAzVmList
		}
		return filterAzGeneric
	case "resource":
		if len(args) > 1 && args[1] == "list" {
			return filterAzResourceList
		}
		return filterAzGeneric
	default:
		return filterAzGeneric
	}
}

// filterAzGeneric compresses any Azure CLI JSON output; preserves errors.
func filterAzGeneric(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAzError(raw) {
		return raw, nil
	}
	compressed, err := compressJSON(raw)
	if err != nil {
		return raw, nil
	}
	return compressed, nil
}

// filterAzVmList extracts name, resourceGroup, state per VM.
func filterAzVmList(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAzError(raw) {
		return raw, nil
	}

	var vms []interface{}
	if err := json.Unmarshal([]byte(raw), &vms); err != nil {
		// Maybe it's a wrapper object
		var data map[string]interface{}
		if err2 := json.Unmarshal([]byte(raw), &data); err2 != nil {
			return raw, nil
		}
		if v, ok := data["value"].([]interface{}); ok {
			vms = v
		} else {
			return compressJSON(raw)
		}
	}

	return formatAzResources(vms, "VMs"), nil
}

// filterAzResourceList extracts name, resourceGroup, type per resource.
func filterAzResourceList(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	if isAzError(raw) {
		return raw, nil
	}

	var resources []interface{}
	if err := json.Unmarshal([]byte(raw), &resources); err != nil {
		return raw, nil
	}

	return formatAzResources(resources, "Resources"), nil
}

func formatAzResources(items []interface{}, label string) string {
	if len(items) == 0 {
		return fmt.Sprintf("No %s found", strings.ToLower(label))
	}

	var lines []string
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := obj["name"].(string)
		rg, _ := obj["resourceGroup"].(string)

		// State can be in powerState, provisioningState, or nested in instanceView
		state := extractAzState(obj)

		line := fmt.Sprintf("%s  %s  %s", name, rg, state)
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return fmt.Sprintf("No %s found", strings.ToLower(label))
	}

	header := fmt.Sprintf("%s (%d):", label, len(lines))
	return header + "\n" + strings.Join(lines, "\n")
}

func extractAzState(obj map[string]interface{}) string {
	// Check powerState directly
	if ps, ok := obj["powerState"].(string); ok && ps != "" {
		return ps
	}
	// Check provisioningState
	if ps, ok := obj["provisioningState"].(string); ok && ps != "" {
		return ps
	}
	// Check nested instanceView.statuses
	if iv, ok := obj["instanceView"].(map[string]interface{}); ok {
		if statuses, ok := iv["statuses"].([]interface{}); ok {
			for _, s := range statuses {
				st, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				if code, _ := st["code"].(string); strings.HasPrefix(code, "PowerState/") {
					return strings.TrimPrefix(code, "PowerState/")
				}
			}
		}
	}
	return "(unknown)"
}

func isAzError(raw string) bool {
	return strings.Contains(raw, "ERROR:") ||
		strings.Contains(raw, "\"error\"") ||
		strings.Contains(raw, "AuthorizationFailed") ||
		strings.Contains(raw, "ResourceNotFound") ||
		strings.Contains(raw, "InvalidAuthenticationToken")
}
