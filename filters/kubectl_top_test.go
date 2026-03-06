package filters

import (
	"strings"
	"testing"
)

func TestFilterKubectlTop(t *testing.T) {
	raw := "NAME                        CPU(cores)   MEMORY(bytes)\n" +
		"web-abc123-xyz              250m         512Mi\n" +
		"api-def456-uvw              100m         256Mi\n"

	got, err := filterKubectlTop(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Function trims input, so compare trimmed
	if got != strings.TrimSpace(raw) {
		t.Errorf("expected passthrough for short output, got: %s", got)
	}
}

func TestFilterKubectlTop_Empty(t *testing.T) {
	got, err := filterKubectlTop("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterKubectlTop_LargePodList(t *testing.T) {
	raw := "NAME                                    CPU(cores)   MEMORY(bytes)   \n" +
		"coredns-5dd5756b68-4xj7t               3m           18Mi            \n" +
		"etcd-minikube                           26m          64Mi            \n" +
		"kube-apiserver-minikube                 58m          271Mi           \n" +
		"kube-controller-manager-minikube        19m          49Mi            \n" +
		"kube-proxy-7wl8m                        1m           17Mi            \n" +
		"kube-scheduler-minikube                 4m           22Mi            \n" +
		"metrics-server-6d94bc8694-kbm4t         5m           15Mi            \n" +
		"nginx-deployment-6b474476c4-2xrqm       1m           3Mi             \n" +
		"nginx-deployment-6b474476c4-7jlkp       1m           4Mi             \n" +
		"nginx-deployment-6b474476c4-xk9jn       1m           3Mi             \n" +
		"redis-master-0                          8m           12Mi            \n" +
		"redis-replica-0                         2m           9Mi             \n" +
		"redis-replica-1                         2m           8Mi             \n" +
		"postgres-primary-0                      15m          128Mi           \n" +
		"postgres-replica-0                      12m          96Mi            \n"

	got, err := filterKubectlTop(raw)
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(raw)
	if got != expected {
		t.Errorf("expected passthrough for large pod list.\ngot:\n%s\nwant:\n%s", got, expected)
	}
	// Verify all 15 data lines are preserved
	lines := strings.Split(got, "\n")
	dataLines := 0
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) != "" {
			dataLines++
		}
	}
	if dataLines != 15 {
		t.Errorf("expected 15 data lines, got %d", dataLines)
	}
}

func TestFilterKubectlTop_NodeMetrics(t *testing.T) {
	raw := "NAME           CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%   \n" +
		"minikube       134m         6%     1247Mi          31%       \n" +
		"worker-node1   89m          4%     892Mi           22%       \n" +
		"worker-node2   156m         7%     1534Mi          38%       \n"

	got, err := filterKubectlTop(raw)
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(raw)
	if got != expected {
		t.Errorf("expected passthrough for node metrics.\ngot:\n%s\nwant:\n%s", got, expected)
	}
	// Verify header contains node-specific columns
	header := strings.Split(got, "\n")[0]
	if !strings.Contains(header, "CPU%") || !strings.Contains(header, "MEMORY%") {
		t.Errorf("expected node header with CPU%% and MEMORY%%, got: %s", header)
	}
}

func TestFilterKubectlTop_ContainersView(t *testing.T) {
	raw := "POD                                     NAME              CPU(cores)   MEMORY(bytes)   \n" +
		"coredns-5dd5756b68-4xj7t               coredns           3m           18Mi            \n" +
		"nginx-deployment-6b474476c4-2xrqm       nginx             1m           3Mi             \n" +
		"nginx-deployment-6b474476c4-2xrqm       istio-proxy       2m           32Mi            \n" +
		"postgres-primary-0                      postgres          15m          128Mi           \n" +
		"postgres-primary-0                      pgbouncer         3m           24Mi            \n"

	got, err := filterKubectlTop(raw)
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(raw)
	if got != expected {
		t.Errorf("expected passthrough for containers view.\ngot:\n%s\nwant:\n%s", got, expected)
	}
	// Verify the POD column header is present (--containers flag output)
	header := strings.Split(got, "\n")[0]
	if !strings.Contains(header, "POD") {
		t.Errorf("expected POD column in containers view header, got: %s", header)
	}
}

func TestFilterKubectlTop_NonTopOutput(t *testing.T) {
	raw := "this is not kubectl top output at all"
	got, err := filterKubectlTop(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("expected raw passthrough for non-top output, got: %s", got)
	}
}
