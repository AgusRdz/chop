package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterGcloudInstancesListTable(t *testing.T) {
	var lines []string
	lines = append(lines, "NAME                   ZONE           MACHINE_TYPE   PREEMPTIBLE  INTERNAL_IP  EXTERNAL_IP     STATUS")
	for i := 0; i < 15; i++ {
		lines = append(lines, fmt.Sprintf("instance-%02d            us-central1-a  e2-medium      false        10.0.0.%d     35.200.0.%d      RUNNING", i, i, i))
	}
	raw := strings.Join(lines, "\n")

	got, err := filterGcloudInstancesList(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain instance names
	if !strings.Contains(got, "instance-00") {
		t.Errorf("expected instance name in output, got:\n%s", got)
	}
	// Should contain zone
	if !strings.Contains(got, "us-central1-a") {
		t.Errorf("expected zone in output, got:\n%s", got)
	}
	// Should contain status
	if !strings.Contains(got, "RUNNING") {
		t.Errorf("expected status in output, got:\n%s", got)
	}

	// Token savings >= 40%
	rawTokens := len(strings.Fields(raw))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 40.0 {
		t.Errorf("expected >=40%% savings, got %.1f%% (raw=%d, filtered=%d)\noutput:\n%s", savings, rawTokens, gotTokens, got)
	}
}

func TestFilterGcloudGenericTableTruncation(t *testing.T) {
	var lines []string
	lines = append(lines, "NAME            LOCATION    STATUS")
	for i := 0; i < 25; i++ {
		lines = append(lines, fmt.Sprintf("resource-%02d     us-east1    ACTIVE", i))
	}
	raw := strings.Join(lines, "\n")

	got, err := filterGcloudGeneric(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should truncate to 10 rows + header + "more" line
	if !strings.Contains(got, "more rows") {
		t.Errorf("expected truncation message, got:\n%s", got)
	}
	// Should show total count
	if !strings.Contains(got, "25 total") {
		t.Errorf("expected total count in truncation message, got:\n%s", got)
	}
}

func TestFilterGcloudJSONOutput(t *testing.T) {
	var items []string
	for i := 0; i < 10; i++ {
		items = append(items, fmt.Sprintf(`{
			"name": "instance-%d",
			"zone": "projects/my-proj/zones/us-central1-a",
			"machineType": "projects/my-proj/zones/us-central1-a/machineTypes/e2-medium",
			"status": "RUNNING",
			"networkInterfaces": [{"networkIP": "10.0.0.%d", "accessConfigs": [{"natIP": "35.200.0.%d"}]}],
			"disks": [{"source": "projects/my-proj/zones/us-central1-a/disks/instance-%d", "type": "PERSISTENT"}],
			"metadata": {"items": [{"key": "startup-script", "value": "#!/bin/bash\necho hello"}]},
			"labels": {"env": "prod", "team": "platform"}
		}`, i, i, i, i))
	}
	raw := "[" + strings.Join(items, ",") + "]"

	got, err := filterGcloudGeneric(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token savings >= 60%
	rawTokens := len(strings.Fields(raw))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 60.0 {
		t.Errorf("expected >=60%% savings, got %.1f%% (raw=%d, filtered=%d)\noutput:\n%s", savings, rawTokens, gotTokens, got)
	}
}

func TestFilterGcloudErrorPreserved(t *testing.T) {
	raw := "ERROR: (gcloud.compute.instances.list) PERMISSION_DENIED: Required 'compute.instances.list' permission for 'projects/my-project'"

	got, err := filterGcloudGeneric(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != raw {
		t.Errorf("expected error preserved, got:\n%s", got)
	}
}

func TestFilterGcloudEmptyOutput(t *testing.T) {
	got, err := filterGcloudGeneric("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty output, got: %q", got)
	}
}

func TestGetGcloudFilter(t *testing.T) {
	if getGcloudFilter(nil) == nil {
		t.Error("expected filterGcloudGeneric for empty args")
	}
	if getGcloudFilter([]string{"unknown"}) == nil {
		t.Error("expected filterGcloudGeneric for unknown subcommand")
	}
	if getGcloudFilter([]string{"compute", "instances", "list"}) == nil {
		t.Error("expected filterGcloudComputeList for compute instances list")
	}
	if getGcloudFilter([]string{"logging", "read"}) == nil {
		t.Error("expected filterGcloudLoggingRead for logging read")
	}
}
