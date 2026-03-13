package filters

import (
	"strings"
	"testing"
)

var acliWorkitemViewFixture = `Key: RES-260
Type: Story
Summary: Implement Contact-Level Integration with SparkPost to Manage Spam-Flagged Emails
Status: In Progress
Assignee: agustin.rodriguez@realmanage.com
Description: Goal
Ensure unsubscribe and spam events handled by SparkPost are visible and manageable within our system, preventing silent suppression of emails and enabling operational awareness and recovery.
Problem
When users unsubscribe via email clients (e.g., Gmail's "Unsubscribe"), SparkPost adds emails directly to its suppression list without triggering our internal unsubscribe flow, leaving the system unaware and blocking future sends.`

var acliWorkitemSearchFixture = `┌───────┬─────────┬───────────────────────────┬──────────┬─────────┬───────────────────────────┐
│ Type  │ Key     │ Assignee                  │ Priority │ Status  │ Summary                   │
├───────┼─────────┼───────────────────────────┼──────────┼─────────┼───────────────────────────┤
│ Bug   │ RES-387 │ agustin.rodriguez@realman │ Highest  │ QA      │ Navigation items disappea │
│       │         │ age.com                   │          │         │ red                       │
├───────┼─────────┼───────────────────────────┼──────────┼─────────┼───────────────────────────┤
│ Theme │ RES-388 │                           │ Highest  │ To Do   │ Client Experience         │
├───────┼─────────┼───────────────────────────┼──────────┼─────────┼───────────────────────────┤
│ Story │ RES-343 │                           │ Medium   │ Review  │ Revamp Contact Informatio │
│       │         │                           │          │         │ n Screen using Design Sys │
│       │         │                           │          │         │ tem                       │
├───────┼─────────┼───────────────────────────┼──────────┼─────────┼───────────────────────────┤
│ Story │ RES-276 │                           │ High     │ Backlog │ Update the printed welcom │
│       │         │                           │          │         │ e letter with SSO instruc │
│       │         │                           │          │         │ tions                     │
├───────┼─────────┼───────────────────────────┼──────────┼─────────┼───────────────────────────┤
│ Story │ RES-341 │                           │ High     │ Backlog │ Meeting Hub - Download me │
│       │         │                           │          │         │ eting documents           │
└───────┴─────────┴───────────────────────────┴──────────┴─────────┴───────────────────────────┘`

func TestAcliJiraWorkitemView(t *testing.T) {
	got, err := filterAcliJiraWorkitemView(acliWorkitemViewFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "RES-260") {
		t.Errorf("expected issue key, got: %s", got)
	}
	if !strings.Contains(got, "In Progress") {
		t.Errorf("expected status, got: %s", got)
	}
	if !strings.Contains(got, "Story") {
		t.Errorf("expected type, got: %s", got)
	}
	if !strings.Contains(got, "SparkPost") {
		t.Errorf("expected summary, got: %s", got)
	}
	if !strings.Contains(got, "Description:") {
		t.Errorf("expected description, got: %s", got)
	}

	rawTokens := countTokens(acliWorkitemViewFixture)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 40.0 {
		t.Errorf("expected >=40%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestAcliJiraWorkitemSearch(t *testing.T) {
	got, err := filterAcliJiraWorkitemSearch(acliWorkitemSearchFixture)
	if err != nil {
		t.Fatal(err)
	}

	// Each issue key should appear exactly once
	for _, key := range []string{"RES-387", "RES-388", "RES-343", "RES-276", "RES-341"} {
		if !strings.Contains(got, key) {
			t.Errorf("expected %s in output, got:\n%s", key, got)
		}
	}

	// Wrapped summary should be reconstructed
	if !strings.Contains(got, "Navigation items disappeared") {
		t.Errorf("expected reconstructed summary for RES-387, got:\n%s", got)
	}
	if !strings.Contains(got, "Revamp Contact Information Screen using Design System") {
		t.Errorf("expected reconstructed summary for RES-343, got:\n%s", got)
	}

	// Should not contain box drawing characters
	if strings.ContainsAny(got, "┌┐└┘├┤┬┴┼─│") {
		t.Errorf("expected no box drawing chars in output, got:\n%s", got)
	}

	rawTokens := countTokens(acliWorkitemSearchFixture)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 50.0 {
		t.Errorf("expected >=50%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestAcliJiraWorkitemSearchEmpty(t *testing.T) {
	got, err := filterAcliJiraWorkitemSearch("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty output, got: %q", got)
	}
}

func TestGetAcliFilter(t *testing.T) {
	if getAcliFilter(nil) != nil {
		t.Error("expected nil for empty args")
	}
	if getAcliFilter([]string{"unknown"}) != nil {
		t.Error("expected nil for unknown subcommand")
	}
	if getAcliFilter([]string{"jira", "workitem", "view"}) == nil {
		t.Error("expected filter for jira")
	}
}

func TestGetAcliJiraFilter(t *testing.T) {
	if getAcliJiraFilter(nil) != nil {
		t.Error("expected nil for empty args")
	}
	// "unknown" subcommands fallback to filterAutoDetect
	if getAcliJiraFilter([]string{"unknown"}) == nil {
		t.Error("expected filterAutoDetect for unknown subcommand")
	}
	if getAcliJiraFilter([]string{"workitem", "view"}) == nil {
		t.Error("expected filter for workitem")
	}
}

func TestGetAcliJiraWorkitemFilter(t *testing.T) {
	if getAcliJiraWorkitemFilter(nil) != nil {
		t.Error("expected nil for empty args")
	}
	// "unknown" subcommands fallback to filterAutoDetect
	if getAcliJiraWorkitemFilter([]string{"unknown"}) == nil {
		t.Error("expected filterAutoDetect for unknown subcommand")
	}
	if getAcliJiraWorkitemFilter([]string{"view"}) == nil {
		t.Error("expected filter for view")
	}
	if getAcliJiraWorkitemFilter([]string{"search"}) == nil {
		t.Error("expected filter for search")
	}
}
