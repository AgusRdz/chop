package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterAwsEc2Describe(t *testing.T) {
	// Build 6 instances across 2 reservations
	var instances []string
	names := []string{"web-1", "web-2", "api-1", "api-2", "worker-1", "db-1"}
	types := []string{"t3.micro", "t3.micro", "t3.medium", "t3.medium", "c5.large", "r5.xlarge"}
	states := []string{"running", "running", "running", "stopped", "running", "running"}
	for i := 0; i < 6; i++ {
		instances = append(instances, fmt.Sprintf(`{
			"InstanceId": "i-0abc%04d",
			"InstanceType": "%s",
			"State": {"Code": 16, "Name": "%s"},
			"Tags": [{"Key": "Name", "Value": "%s"}, {"Key": "Env", "Value": "prod"}],
			"LaunchTime": "2024-01-15T10:30:00+00:00",
			"PrivateIpAddress": "10.0.%d.%d",
			"PublicIpAddress": "54.200.%d.%d",
			"SubnetId": "subnet-abc123",
			"VpcId": "vpc-def456",
			"SecurityGroups": [{"GroupName": "default", "GroupId": "sg-123"}],
			"Architecture": "x86_64",
			"Hypervisor": "xen",
			"RootDeviceType": "ebs"
		}`, i, types[i], states[i], names[i], i, i, i, i))
	}

	raw := fmt.Sprintf(`{
		"Reservations": [
			{"ReservationId": "r-001", "Instances": [%s]},
			{"ReservationId": "r-002", "Instances": [%s]}
		]
	}`, strings.Join(instances[:3], ","), strings.Join(instances[3:], ","))

	got, err := filterAwsEc2Describe(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain instance IDs
	if !strings.Contains(got, "i-0abc0000") {
		t.Errorf("expected instance ID in output, got:\n%s", got)
	}
	// Should contain state
	if !strings.Contains(got, "running") {
		t.Errorf("expected state in output, got:\n%s", got)
	}
	// Should contain name from tags
	if !strings.Contains(got, "web-1") {
		t.Errorf("expected Name tag in output, got:\n%s", got)
	}
	// Should contain instance type
	if !strings.Contains(got, "t3.micro") {
		t.Errorf("expected instance type in output, got:\n%s", got)
	}
	// Should show count
	if !strings.Contains(got, "6") {
		t.Errorf("expected instance count in output, got:\n%s", got)
	}

	// Token savings >= 70%
	rawTokens := len(strings.Fields(raw))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 70.0 {
		t.Errorf("expected >=70%% savings, got %.1f%% (raw=%d, filtered=%d)\noutput:\n%s", savings, rawTokens, gotTokens, got)
	}
}

func TestFilterAwsS3Ls(t *testing.T) {
	var lines []string
	// 10 files under "logs/" prefix
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-15 10:30:00      %d logs/app-%d.log", (i+1)*1024, i))
	}
	// 8 files under "data/" prefix
	for i := 0; i < 8; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-15 10:30:00      %d data/export-%d.csv", (i+1)*2048, i))
	}
	// 5 files under "backup/" prefix
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-15 10:30:00      %d backup/snap-%d.tar.gz", (i+1)*10240, i))
	}
	raw := strings.Join(lines, "\n")

	got, err := filterAwsS3Ls(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should group by prefix
	if !strings.Contains(got, "logs/") {
		t.Errorf("expected 'logs/' prefix in output, got:\n%s", got)
	}
	if !strings.Contains(got, "data/") {
		t.Errorf("expected 'data/' prefix in output, got:\n%s", got)
	}
	if !strings.Contains(got, "10 files") {
		t.Errorf("expected '10 files' count in output, got:\n%s", got)
	}

	// Token savings >= 50%
	rawTokens := len(strings.Fields(raw))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 50.0 {
		t.Errorf("expected >=50%% savings, got %.1f%% (raw=%d, filtered=%d)\noutput:\n%s", savings, rawTokens, gotTokens, got)
	}
}

func TestFilterAwsErrorPreserved(t *testing.T) {
	raw := "An error occurred (AccessDenied) when calling the DescribeInstances operation: User: arn:aws:iam::123456789012:user/dev is not authorized to perform: ec2:DescribeInstances"

	got, err := filterAwsEc2Describe(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != raw {
		t.Errorf("expected error preserved, got:\n%s", got)
	}
}

func TestFilterAwsGenericJSON(t *testing.T) {
	// Large generic JSON
	var items []string
	for i := 0; i < 20; i++ {
		items = append(items, fmt.Sprintf(`{"id": "item-%d", "name": "Item %d", "status": "active", "created": "2024-01-15T10:30:00Z", "region": "us-east-1"}`, i, i))
	}
	raw := `{"Items": [` + strings.Join(items, ",") + `]}`

	got, err := filterAwsGeneric(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be compressed (smaller than raw)
	if len(got) >= len(raw) {
		t.Errorf("expected compressed output smaller than raw (%d >= %d)\noutput:\n%s", len(got), len(raw), got)
	}
}

func TestFilterAwsLogs(t *testing.T) {
	// Simulate aws logs get-log-events output with duplicate messages
	events := `{
		"events": [
			{"timestamp": 1705312200000, "message": "2024-01-15T10:30:00.000Z Starting application"},
			{"timestamp": 1705312201000, "message": "2024-01-15T10:30:01.000Z Health check passed"},
			{"timestamp": 1705312202000, "message": "2024-01-15T10:30:02.000Z Health check passed"},
			{"timestamp": 1705312203000, "message": "2024-01-15T10:30:03.000Z Health check passed"},
			{"timestamp": 1705312204000, "message": "2024-01-15T10:30:04.000Z Health check passed"},
			{"timestamp": 1705312205000, "message": "2024-01-15T10:30:05.000Z Request processed"},
			{"timestamp": 1705312206000, "message": "2024-01-15T10:30:06.000Z Health check passed"},
			{"timestamp": 1705312207000, "message": "2024-01-15T10:30:07.000Z Request processed"},
			{"timestamp": 1705312208000, "message": "2024-01-15T10:30:08.000Z Shutting down"}
		],
		"nextForwardToken": "f/abc123"
	}`

	got, err := filterAwsLogs(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should deduplicate "Health check passed"
	if strings.Count(got, "Health check passed") != 1 {
		t.Errorf("expected deduplicated health check messages, got:\n%s", got)
	}
	// Should show count
	if !strings.Contains(got, "x5") {
		t.Errorf("expected [x5] for repeated health checks, got:\n%s", got)
	}
	// Should show unique count
	if !strings.Contains(got, "unique") {
		t.Errorf("expected unique count in header, got:\n%s", got)
	}
}

func TestGetAwsFilter(t *testing.T) {
	if getAwsFilter(nil) == nil {
		t.Error("expected filterAwsGeneric for empty args")
	}
	if getAwsFilter([]string{"unknown"}) == nil {
		t.Error("expected filterAwsGeneric for unknown subcommand")
	}
	if getAwsFilter([]string{"s3", "ls"}) == nil {
		t.Error("expected filterAwsS3Ls for s3 ls")
	}
	if getAwsFilter([]string{"ec2", "describe-instances"}) == nil {
		t.Error("expected filterAwsEc2Describe for ec2 describe-instances")
	}
	if getAwsFilter([]string{"logs"}) == nil {
		t.Error("expected filterAwsLogs for logs")
	}
}
