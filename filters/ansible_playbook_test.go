package filters

import (
	"strings"
	"testing"
)

func TestFilterAnsiblePlaybook(t *testing.T) {
	raw := `
PLAY [Install Web Servers] ******************************************************

TASK [Gathering Facts] *********************************************************
ok: [web1]
ok: [web2]

TASK [Install Nginx] ***********************************************************
changed: [web1]
ok: [web2]

TASK [Copy configuration file] *************************************************
skipping: [web1]
skipping: [web2]

TASK [Start Nginx] *************************************************************
fatal: [web1]: FAILED! => {"changed": false, "msg": "Service failed to start"}
ok: [web2]

PLAY RECAP *********************************************************************
web1                       : ok=1    changed=1    unreachable=0    failed=1    skipped=1    rescued=0    ignored=0
web2                       : ok=3    changed=0    unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
`

	got, err := filterAnsiblePlaybook(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not have the "ok: [web1]"
	if strings.Contains(got, "ok: [web1]") {
		t.Errorf("expected 'ok: [web1]' to be filtered out, got:\n%s", got)
	}

	// Should have the "changed: [web1]"
	if !strings.Contains(got, "changed: [web1]") {
		t.Errorf("expected 'changed: [web1]' to be included, got:\n%s", got)
	}

	// Should have the "fatal: [web1]: FAILED!"
	if !strings.Contains(got, "fatal: [web1]: FAILED!") {
		t.Errorf("expected 'fatal: [web1]: FAILED!' to be included, got:\n%s", got)
	}

	// Should have the PLAY RECAP
	if !strings.Contains(got, "PLAY RECAP *****************") {
		t.Errorf("expected 'PLAY RECAP' to be included, got:\n%s", got)
	}

	// Should not include "Gathering Facts" task since all were "ok"
	if strings.Contains(got, "TASK [Gathering Facts]") {
		t.Errorf("expected 'Gathering Facts' task header to be filtered out, got:\n%s", got)
	}

	// Should include "Install Nginx" task since it had a "changed"
	if !strings.Contains(got, "TASK [Install Nginx]") {
		t.Errorf("expected 'Install Nginx' task header to be included, got:\n%s", got)
	}
}
