package filters

import (
	"strings"
	"testing"
)

func TestFilterHelmInstall(t *testing.T) {
	raw := `Release "my-release" has been installed. Happy Helming!
NAME: my-release
LAST DEPLOYED: Mon Jan 15 10:30:00 2024
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods)
  echo "Visit http://127.0.0.1:8080"
  kubectl --namespace default port-forward $POD_NAME 8080:80`

	got, err := filterHelmInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
	if !strings.Contains(got, "my-release") {
		t.Error("expected release name")
	}
	if !strings.Contains(got, "deployed") {
		t.Error("expected status")
	}
	if strings.Contains(got, "kubectl") {
		t.Error("NOTES should be stripped")
	}
}

func TestFilterHelmList(t *testing.T) {
	raw := "NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION\n" +
		"my-release      default         3               2024-01-15 10:30:00.000 +0000 UTC       deployed        myapp-1.0.0     1.0.0\n" +
		"other-release   staging         1               2024-01-14 08:00:00.000 +0000 UTC       deployed        otherapp-2.0    2.0.0\n"

	got, err := filterHelmList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "2 releases") {
		t.Errorf("expected release count, got: %s", got)
	}
}

func TestFilterHelmInstall_Empty(t *testing.T) {
	got, err := filterHelmInstall("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterHelmInstall_RealisticInstall(t *testing.T) {
	raw := `NAME: my-release
LAST DEPLOYED: Thu Mar  6 10:30:45 2026
NAMESPACE: production
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods --namespace production -l "app.kubernetes.io/name=nginx,app.kubernetes.io/instance=my-release" -o jsonpath="{.items[0].metadata.name}")
  export CONTAINER_PORT=$(kubectl get pod --namespace production $POD_NAME -o jsonpath="{.spec.containers[0].ports[0].containerPort}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl --namespace production port-forward $POD_NAME 8080:$CONTAINER_PORT

2. Verify the deployment:
  kubectl get pods -n production
  kubectl get svc -n production

WARNING: Kubernetes configuration file is group-readable. This is insecure.`

	got, err := filterHelmInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got len %d >= %d", len(got), len(raw))
	}
	if !strings.Contains(got, "my-release") {
		t.Error("expected release name")
	}
	if !strings.Contains(got, "deployed") {
		t.Error("expected status")
	}
	if !strings.Contains(got, "revision 1") {
		t.Error("expected revision")
	}
	if !strings.Contains(got, "production") {
		t.Error("expected namespace")
	}
	if strings.Contains(got, "kubectl") {
		t.Error("NOTES section should be stripped")
	}
	if strings.Contains(got, "WARNING") {
		t.Error("WARNING after NOTES should be stripped")
	}
	if strings.Contains(got, "port-forward") {
		t.Error("port-forward instructions should be stripped")
	}
}

func TestFilterHelmInstall_RealisticUpgrade(t *testing.T) {
	raw := `Release "my-release" has been upgraded. Happy Helming!
NAME: my-release
LAST DEPLOYED: Thu Mar  6 10:35:22 2026
NAMESPACE: production
STATUS: deployed
REVISION: 2
NOTES:
Application has been upgraded successfully.
Run 'kubectl get pods -n production' to verify.`

	got, err := filterHelmInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got len %d >= %d", len(got), len(raw))
	}
	if !strings.Contains(got, "my-release") {
		t.Error("expected release name")
	}
	if !strings.Contains(got, "deployed") {
		t.Error("expected status")
	}
	if !strings.Contains(got, "revision 2") {
		t.Error("expected revision 2")
	}
	if !strings.Contains(got, "production") {
		t.Error("expected namespace")
	}
	if strings.Contains(got, "upgraded successfully") {
		t.Error("NOTES content should be stripped")
	}
	if strings.Contains(got, "Happy Helming") {
		t.Error("banner line should not appear in compressed output")
	}
}

func TestFilterHelmList_RealisticTenReleases(t *testing.T) {
	raw := "NAME          	NAMESPACE  	REVISION	UPDATED                                	STATUS  	CHART              	APP VERSION\n" +
		"cert-manager  	cert-mgr   	3       	2026-02-15 08:30:12.123456 +0000 UTC   	deployed	cert-manager-1.14.2	1.14.2     \n" +
		"ingress-nginx 	ingress    	5       	2026-03-01 14:22:33.654321 +0000 UTC   	deployed	ingress-nginx-4.9.0	1.9.5      \n" +
		"prometheus    	monitoring 	2       	2026-01-20 11:15:44.789012 +0000 UTC   	deployed	prometheus-25.8.2  	2.49.1     \n" +
		"grafana       	monitoring 	4       	2026-02-28 09:45:01.234567 +0000 UTC   	deployed	grafana-7.3.3      	10.3.1     \n" +
		"redis         	default    	1       	2026-03-05 16:00:55.345678 +0000 UTC   	deployed	redis-18.6.1       	7.2.4      \n" +
		"postgres      	database   	6       	2026-02-10 07:30:22.456789 +0000 UTC   	deployed	postgresql-14.0.5  	16.2.0     \n" +
		"rabbitmq      	messaging  	2       	2026-03-02 13:20:11.567890 +0000 UTC   	deployed	rabbitmq-13.0.3    	3.13.0     \n" +
		"vault         	security   	3       	2026-01-15 10:10:33.678901 +0000 UTC   	deployed	vault-0.27.0       	1.15.4     \n" +
		"argocd        	argocd     	7       	2026-03-04 15:55:44.789012 +0000 UTC   	deployed	argo-cd-5.53.12    	2.10.1     \n" +
		"elasticsearch 	logging    	2       	2026-02-20 12:40:55.890123 +0000 UTC   	deployed	elasticsearch-8.5.1	8.12.1     \n"

	got, err := filterHelmList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got len %d >= %d", len(got), len(raw))
	}
	if !strings.Contains(got, "10 releases") {
		t.Errorf("expected '10 releases' count, got: %s", got)
	}
	// Verify some release names are present
	for _, name := range []string{"cert-manager", "ingress-nginx", "prometheus", "grafana", "redis", "postgres", "rabbitmq", "vault", "argocd", "elasticsearch"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected release %q in output", name)
		}
	}
	// Verify namespaces are included
	for _, ns := range []string{"cert-mgr", "ingress", "monitoring", "default", "database", "messaging", "security", "argocd", "logging"} {
		if !strings.Contains(got, ns) {
			t.Errorf("expected namespace %q in output", ns)
		}
	}
	// Verify chart names are present
	if !strings.Contains(got, "cert-manager-1.14.2") {
		t.Error("expected chart name in output")
	}
	// Verify timestamps are stripped
	if strings.Contains(got, "2026-02-15") {
		t.Error("timestamps should be stripped")
	}
	// Verify status is present
	if !strings.Contains(got, "deployed") {
		t.Error("expected status in output")
	}
}

func TestFilterHelmList_Empty(t *testing.T) {
	got, err := filterHelmList("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterHelmList_HeaderOnly(t *testing.T) {
	raw := "NAME\tNAMESPACE\tREVISION\tUPDATED\tSTATUS\tCHART\tAPP VERSION\n"

	got, err := filterHelmList(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Header-only (1 line) passes through as-is
	if got != strings.TrimSpace(raw) {
		t.Errorf("expected header passthrough, got %q", got)
	}
}
