package filters

import (
	"strings"
	"testing"
)

func TestFilterKubectlGetPods(t *testing.T) {
	raw := `NAME                                    READY   STATUS    RESTARTS   AGE   IP            NODE                 NOMINATED NODE   READINESS GATES
api-server-7d9b6c8f5-abc12              1/1     Running   0          2d    10.244.1.5    worker-1             <none>           <none>
api-server-7d9b6c8f5-def34              1/1     Running   0          2d    10.244.2.8    worker-2             <none>           <none>
cache-redis-0                           1/1     Running   2          5d    10.244.1.10   worker-1             <none>           <none>
celery-worker-6f8d4c7b9-ghi56          1/1     Running   0          1d    10.244.3.2    worker-3             <none>           <none>
frontend-5c8e7a3d1-jkl78               1/1     Running   0          3h    10.244.2.15   worker-2             <none>           <none>
ingress-nginx-controller-mn901          1/1     Running   1          10d   10.244.1.20   worker-1             <none>           <none>
monitoring-grafana-8b7c6d5e4-opq23     1/1     Running   0          7d    10.244.3.8    worker-3             <none>           <none>
postgres-primary-0                      1/1     Running   0          5d    10.244.1.25   worker-1             <none>           <none>`

	got, err := filterKubectlGet(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain NAME, STATUS, RESTARTS, AGE columns
	if !strings.Contains(got, "NAME") {
		t.Error("expected NAME column in output")
	}
	if !strings.Contains(got, "STATUS") {
		t.Error("expected STATUS column in output")
	}
	if !strings.Contains(got, "RESTARTS") {
		t.Error("expected RESTARTS column in output")
	}

	// Should NOT contain IP, NODE columns
	if strings.Contains(got, "10.244.1.5") {
		t.Error("expected IP addresses to be stripped")
	}
	if strings.Contains(got, "worker-1") {
		t.Error("expected NODE column to be stripped")
	}
	if strings.Contains(got, "NOMINATED") {
		t.Error("expected NOMINATED NODE column to be stripped")
	}

	// Should have 9 lines (header + 8 pods)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 9 {
		t.Errorf("expected 9 lines, got %d:\n%s", len(lines), got)
	}

	// Token savings >= 40%
	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 40.0 {
		t.Errorf("expected >=40%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
}

func TestFilterKubectlGetServices(t *testing.T) {
	raw := `NAME                  TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                      AGE    SELECTOR
kubernetes            ClusterIP      10.96.0.1        <none>        443/TCP                      30d    <none>
api-service           ClusterIP      10.96.45.12      <none>        8080/TCP                     5d     app=api
frontend-service      LoadBalancer   10.96.78.34      34.56.78.90   80:31234/TCP,443:31235/TCP   3d     app=frontend
redis-service         ClusterIP      10.96.90.56      <none>        6379/TCP                     5d     app=redis
postgres-service      ClusterIP      10.96.12.78      <none>        5432/TCP                     5d     app=postgres`

	got, err := filterKubectlGet(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain NAME, TYPE, CLUSTER-IP, PORT(S)
	if !strings.Contains(got, "TYPE") {
		t.Error("expected TYPE column in output")
	}
	if !strings.Contains(got, "CLUSTER-IP") {
		t.Error("expected CLUSTER-IP column in output")
	}
	if !strings.Contains(got, "PORT(S)") {
		t.Error("expected PORT(S) column in output")
	}

	// Should NOT contain AGE or SELECTOR
	if strings.Contains(got, "SELECTOR") {
		t.Error("expected SELECTOR column to be stripped")
	}

	// Token savings >= 30%
	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 30.0 {
		t.Errorf("expected >=30%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
}

func TestFilterKubectlGetNoResources(t *testing.T) {
	raw := `No resources found in default namespace.`

	got, err := filterKubectlGet(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "No resources found") {
		t.Errorf("expected 'No resources found' message, got: %s", got)
	}
}

func TestFilterKubectlGetEmpty(t *testing.T) {
	got, err := filterKubectlGet("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "No resources found" {
		t.Errorf("expected 'No resources found', got: %s", got)
	}
}

func TestFilterKubectlGetJSONPassthrough(t *testing.T) {
	raw := `{"apiVersion": "v1", "kind": "PodList", "items": []}`

	got, err := filterKubectlGet(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should now be pretty printed and compressed by compressJSON
	expected, _ := compressJSON(raw)
	if got != expected {
		t.Errorf("expected compressed JSON, got:\n%s\nexpected:\n%s", got, expected)
	}
}
