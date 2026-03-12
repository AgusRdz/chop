package filters

import (
	"strings"
	"testing"
)

var dockerRmiFixture = `Untagged: myimage:latest
Untagged: myimage@sha256:abc123def456789abc123def456789abc123def456789abc123def456789abcdef
Deleted: sha256:abc123def456789abc123def456789abc123def456789abc123def456789abcdef
Deleted: sha256:def456abc123789def456abc123789def456abc123789def456abc123789def456
Deleted: sha256:789012345678901234567890123456789012345678901234567890123456789012`

func TestDockerRmiStripsDeletedLayers(t *testing.T) {
	got, err := filterDockerRmi(dockerRmiFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "Untagged: myimage:latest") {
		t.Errorf("expected untagged line in output: %s", got)
	}
	if strings.Contains(got, "Deleted: sha256:") {
		t.Errorf("Deleted sha256 lines should be stripped: %s", got)
	}

	rawTokens := countTokens(dockerRmiFixture)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 50.0 {
		t.Errorf("expected >=50%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestDockerRmiRouted(t *testing.T) {
	f := getDockerFilter([]string{"rmi"})
	if f == nil {
		t.Fatal("expected filter for docker rmi, got nil")
	}
}
