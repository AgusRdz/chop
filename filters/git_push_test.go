package filters

import (
	"strings"
	"testing"
)

var gitPushFixture = `Enumerating objects: 5, done.
Counting objects: 100% (5/5), done.
Delta compression using up to 8 threads
Compressing objects: 100% (3/3), done.
Writing objects: 100% (3/3), 1.23 KiB | 1.23 MiB/s, done.
Total 3 (delta 2), reused 0 (delta 0), pack-reused 0
remote: Resolving deltas: 100% (2/2), completed with 2 local objects.
To https://github.com/user/repo.git
   abc1234..def5678  main -> main`

var gitPushNewBranchFixture = `Total 0 (delta 0), reused 0 (delta 0), pack-reused 0
remote:
remote: Create a pull request for 'feature/my-branch' on GitHub by visiting:
remote:      https://github.com/user/repo/pull/new/feature/my-branch
remote:
To https://github.com/user/repo.git
 * [new branch]      feature/my-branch -> feature/my-branch`

var gitPushTagFixture = `Total 0 (delta 0), reused 0 (delta 0), pack-reused 0
To https://github.com/user/repo.git
 * [new tag]         v1.5.0 -> v1.5.0`

func TestGitPushStripsProgress(t *testing.T) {
	got, err := filterGitPush(gitPushFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "To https://github.com") {
		t.Errorf("expected remote URL in output: %s", got)
	}
	if !strings.Contains(got, "main -> main") {
		t.Errorf("expected branch ref in output: %s", got)
	}

	// Noise stripped
	if strings.Contains(got, "Enumerating") {
		t.Errorf("Enumerating should be stripped: %s", got)
	}
	if strings.Contains(got, "Counting") {
		t.Errorf("Counting should be stripped: %s", got)
	}
	if strings.Contains(got, "Compressing") {
		t.Errorf("Compressing should be stripped: %s", got)
	}
	if strings.Contains(got, "Writing objects") {
		t.Errorf("Writing objects should be stripped: %s", got)
	}
	if strings.Contains(got, "Resolving deltas") {
		t.Errorf("Resolving deltas should be stripped: %s", got)
	}

	rawTokens := countTokens(gitPushFixture)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 50.0 {
		t.Errorf("expected >=50%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestGitPushNewBranch(t *testing.T) {
	got, err := filterGitPush(gitPushNewBranchFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "To https://github.com") {
		t.Errorf("expected remote URL: %s", got)
	}
	if !strings.Contains(got, "[new branch]") {
		t.Errorf("expected new branch marker: %s", got)
	}
	// PR URL in remote: lines should be kept
	if !strings.Contains(got, "pull/new") {
		t.Errorf("expected PR URL in remote lines: %s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestGitPushTag(t *testing.T) {
	got, err := filterGitPush(gitPushTagFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "[new tag]") {
		t.Errorf("expected new tag marker: %s", got)
	}
	if !strings.Contains(got, "v1.5.0") {
		t.Errorf("expected tag name: %s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestGitPushUpToDate(t *testing.T) {
	raw := "Everything up-to-date"
	got, err := filterGitPush(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("up-to-date message should pass through: %s", got)
	}
}

func TestGitPushRouted(t *testing.T) {
	f := getGitFilter([]string{"push"})
	if f == nil {
		t.Fatal("expected filter for git push, got nil")
	}
}
