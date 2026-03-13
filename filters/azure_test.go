package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterAzVmList(t *testing.T) {
	var vms []string
	for i := 0; i < 6; i++ {
		vms = append(vms, fmt.Sprintf(`{
			"name": "vm-%d",
			"resourceGroup": "rg-prod",
			"location": "eastus",
			"powerState": "VM running",
			"provisioningState": "Succeeded",
			"hardwareProfile": {"vmSize": "Standard_D2s_v3"},
			"storageProfile": {"osDisk": {"name": "osdisk-%d", "diskSizeGb": 128}},
			"networkProfile": {"networkInterfaces": [{"id": "/subscriptions/sub-1/resourceGroups/rg-prod/providers/Microsoft.Network/networkInterfaces/nic-%d"}]},
			"osProfile": {"computerName": "vm-%d", "adminUsername": "azureuser"},
			"id": "/subscriptions/sub-1/resourceGroups/rg-prod/providers/Microsoft.Compute/virtualMachines/vm-%d",
			"type": "Microsoft.Compute/virtualMachines",
			"tags": {"env": "prod", "team": "platform"}
		}`, i, i, i, i, i))
	}
	raw := "[" + strings.Join(vms, ",") + "]"

	got, err := filterAzVmList(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain VM names
	if !strings.Contains(got, "vm-0") {
		t.Errorf("expected VM name in output, got:\n%s", got)
	}
	// Should contain resource group
	if !strings.Contains(got, "rg-prod") {
		t.Errorf("expected resource group in output, got:\n%s", got)
	}
	// Should contain state
	if !strings.Contains(got, "VM running") {
		t.Errorf("expected power state in output, got:\n%s", got)
	}
	// Should show count
	if !strings.Contains(got, "6") {
		t.Errorf("expected VM count in output, got:\n%s", got)
	}

	// Token savings >= 60%
	rawTokens := len(strings.Fields(raw))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 60.0 {
		t.Errorf("expected >=60%% savings, got %.1f%% (raw=%d, filtered=%d)\noutput:\n%s", savings, rawTokens, gotTokens, got)
	}
}

func TestFilterAzErrorPreserved(t *testing.T) {
	raw := `ERROR: AuthorizationFailed: The client 'user@example.com' with object id 'abc-123' does not have authorization to perform action 'Microsoft.Compute/virtualMachines/read' over scope '/subscriptions/sub-1'.`

	got, err := filterAzVmList(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != raw {
		t.Errorf("expected error preserved, got:\n%s", got)
	}
}

func TestFilterAzGenericJSON(t *testing.T) {
	var items []string
	for i := 0; i < 15; i++ {
		items = append(items, fmt.Sprintf(`{
			"name": "resource-%d",
			"resourceGroup": "rg-prod",
			"type": "Microsoft.Storage/storageAccounts",
			"location": "eastus",
			"provisioningState": "Succeeded",
			"sku": {"name": "Standard_LRS", "tier": "Standard"},
			"kind": "StorageV2",
			"id": "/subscriptions/sub-1/resourceGroups/rg-prod/providers/Microsoft.Storage/storageAccounts/resource-%d"
		}`, i, i))
	}
	raw := "[" + strings.Join(items, ",") + "]"

	got, err := filterAzGeneric(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be compressed
	if len(got) >= len(raw) {
		t.Errorf("expected compressed output, raw=%d filtered=%d\noutput:\n%s", len(raw), len(got), got)
	}
}

func TestFilterAzResourceList(t *testing.T) {
	var resources []string
	for i := 0; i < 5; i++ {
		resources = append(resources, fmt.Sprintf(`{
			"name": "res-%d",
			"resourceGroup": "rg-dev",
			"provisioningState": "Succeeded",
			"type": "Microsoft.Web/sites",
			"location": "westus2",
			"id": "/subscriptions/sub-1/resourceGroups/rg-dev/providers/Microsoft.Web/sites/res-%d"
		}`, i, i))
	}
	raw := "[" + strings.Join(resources, ",") + "]"

	got, err := filterAzResourceList(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "res-0") {
		t.Errorf("expected resource name in output, got:\n%s", got)
	}
	if !strings.Contains(got, "rg-dev") {
		t.Errorf("expected resource group in output, got:\n%s", got)
	}
	if !strings.Contains(got, "5") {
		t.Errorf("expected count in output, got:\n%s", got)
	}
}

func TestGetAzFilter(t *testing.T) {
	if getAzFilter(nil) == nil {
		t.Error("expected filterAzGeneric for empty args")
	}
	if getAzFilter([]string{"unknown"}) == nil {
		t.Error("expected filterAzGeneric for unknown subcommand")
	}
	if getAzFilter([]string{"vm", "list"}) == nil {
		t.Error("expected filterAzVmList for vm list")
	}
}
