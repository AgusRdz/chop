package tracking

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	initForTest()
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "test_tracking.db")
	t.Setenv("CHOP_DB_PATH", dbFile)
}

func TestInitCreatesDBAndTable(t *testing.T) {
	setupTestDB(t)

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	path := os.Getenv("CHOP_DB_PATH")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestTrackAndGetStats(t *testing.T) {
	setupTestDB(t)

	// Track 5 commands with varying savings
	commands := []struct {
		cmd      string
		raw, fil int
	}{
		{"git status", 100, 20},
		{"git log", 200, 40},
		{"docker ps", 80, 30},
		{"npm list", 150, 45},
		{"git diff", 300, 60},
	}

	for _, c := range commands {
		if err := Track(c.cmd, c.raw, c.fil); err != nil {
			t.Fatalf("Track(%s) failed: %v", c.cmd, err)
		}
	}

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalCommands != 5 {
		t.Errorf("expected 5 commands, got %d", stats.TotalCommands)
	}

	expectedRaw := 100 + 200 + 80 + 150 + 300
	if stats.TotalRawTokens != expectedRaw {
		t.Errorf("expected %d raw tokens, got %d", expectedRaw, stats.TotalRawTokens)
	}

	expectedSaved := (100 - 20) + (200 - 40) + (80 - 30) + (150 - 45) + (300 - 60)
	if stats.TotalSavedTokens != expectedSaved {
		t.Errorf("expected %d saved tokens, got %d", expectedSaved, stats.TotalSavedTokens)
	}

	expectedPct := float64(expectedSaved) / float64(expectedRaw) * 100.0
	if diff := stats.OverallSavingsPct - expectedPct; diff > 0.1 || diff < -0.1 {
		t.Errorf("expected %.1f%% savings, got %.1f%%", expectedPct, stats.OverallSavingsPct)
	}

	// Today's stats should match total (all tracked today)
	if stats.TodayCommands != 5 {
		t.Errorf("expected 5 today commands, got %d", stats.TodayCommands)
	}
	if stats.TodaySavedTokens != expectedSaved {
		t.Errorf("expected %d today saved, got %d", expectedSaved, stats.TodaySavedTokens)
	}
}

func TestGetHistoryReverseChronological(t *testing.T) {
	setupTestDB(t)

	commands := []string{"git status", "git log", "docker ps"}
	for _, cmd := range commands {
		if err := Track(cmd, 100, 50); err != nil {
			t.Fatalf("Track(%s) failed: %v", cmd, err)
		}
	}

	records, err := GetHistory(10)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	// Most recent first
	if records[0].Command != "docker ps" {
		t.Errorf("expected first record 'docker ps', got %q", records[0].Command)
	}
	if records[2].Command != "git status" {
		t.Errorf("expected last record 'git status', got %q", records[2].Command)
	}
}

func TestGetHistoryLimit(t *testing.T) {
	setupTestDB(t)

	for i := 0; i < 10; i++ {
		if err := Track("cmd", 100, 50); err != nil {
			t.Fatal(err)
		}
	}

	records, err := GetHistory(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
}

func TestCleanupRemovesOldRecords(t *testing.T) {
	setupTestDB(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Insert an old record directly
	old := time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO tracking (timestamp, command, raw_tokens, filtered_tokens, savings_pct) VALUES (?, ?, ?, ?, ?)`,
		old, "old cmd", 100, 50, 50.0,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a recent record
	if err := Track("new cmd", 100, 50); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(90); err != nil {
		t.Fatal(err)
	}

	records, err := GetHistory(100)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record after cleanup, got %d", len(records))
	}
	if records[0].Command != "new cmd" {
		t.Errorf("expected 'new cmd', got %q", records[0].Command)
	}
}

func TestTrackSilentOnZeroRaw(t *testing.T) {
	setupTestDB(t)

	// Should not panic or error on zero raw tokens
	if err := Track("empty", 0, 0); err != nil {
		t.Fatalf("Track with zero tokens failed: %v", err)
	}

	stats, err := GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCommands != 1 {
		t.Errorf("expected 1 command, got %d", stats.TotalCommands)
	}
}

func TestCountTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"  spaces  everywhere  ", 2},
		{"one", 1},
		{"", 0},
		{"multi\nline\ntext", 3},
	}
	for _, tc := range tests {
		got := CountTokens(tc.input)
		if got != tc.expected {
			t.Errorf("CountTokens(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestFormatGain(t *testing.T) {
	s := Stats{
		TotalCommands:     1203,
		TotalRawTokens:    500000,
		TotalSavedTokens:  456789,
		OverallSavingsPct: 78.3,
		TodayCommands:     45,
		TodaySavedTokens:  12340,
		WeekCommands:      210,
		WeekSavedTokens:   98765,
		MonthCommands:     890,
		MonthSavedTokens:  345678,
		YearCommands:      1150,
		YearSavedTokens:   430000,
	}
	out := FormatGain(s)
	if !strings.Contains(out, "45 commands") {
		t.Errorf("missing today commands in output: %s", out)
	}
	if !strings.Contains(out, "12,340") {
		t.Errorf("missing formatted today saved in output: %s", out)
	}
	if !strings.Contains(out, "210 commands") {
		t.Errorf("missing week commands in output: %s", out)
	}
	if !strings.Contains(out, "890 commands") {
		t.Errorf("missing month commands in output: %s", out)
	}
	if !strings.Contains(out, "1150 commands") {
		t.Errorf("missing year commands in output: %s", out)
	}
	if !strings.Contains(out, "78.3%") {
		t.Errorf("missing overall pct in output: %s", out)
	}
}

func TestFormatHistory(t *testing.T) {
	records := []Record{
		{Timestamp: "2026-03-05 14:23:00", Command: "git status", RawTokens: 67, FilteredTokens: 8, SavingsPct: 88.1},
	}
	out := FormatHistory(records)
	if !strings.Contains(out, "git status") {
		t.Errorf("missing command in history: %s", out)
	}
	if !strings.Contains(out, "88.1%") {
		t.Errorf("missing savings in history: %s", out)
	}
}

func TestGetUnchopped(t *testing.T) {
	setupTestDB(t)

	// Commands with savings (should NOT appear)
	if err := Track("git status", 100, 20); err != nil {
		t.Fatal(err)
	}
	if err := Track("docker ps", 80, 30); err != nil {
		t.Fatal(err)
	}

	// Commands with 0% savings (should appear regardless of token count)
	if err := Track("ls -la /tmp", 50, 50); err != nil {
		t.Fatal(err)
	}
	if err := Track("ls -la /home", 60, 60); err != nil {
		t.Fatal(err)
	}
	if err := Track("ls -la /var", 40, 40); err != nil {
		t.Fatal(err)
	}
	if err := Track("whoami", 10, 10); err != nil {
		t.Fatal(err)
	}

	// Zero raw tokens should be excluded
	if err := Track("empty", 0, 0); err != nil {
		t.Fatal(err)
	}

	results, err := GetUnchopped()
	if err != nil {
		t.Fatalf("GetUnchopped failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 unchopped commands, got %d", len(results))
	}

	// "ls -la" should be first (3 calls > 1 call for "whoami")
	if results[0].Command != "ls -la" {
		t.Errorf("expected first command 'ls -la', got %q", results[0].Command)
	}
	if results[0].Count != 3 {
		t.Errorf("expected 3 calls for ls -la, got %d", results[0].Count)
	}
	if results[0].TotalTokens != 150 {
		t.Errorf("expected 150 total tokens for ls -la, got %d", results[0].TotalTokens)
	}

	if results[1].Command != "whoami" {
		t.Errorf("expected second command 'whoami', got %q", results[1].Command)
	}
	if results[1].Count != 1 {
		t.Errorf("expected 1 call for whoami, got %d", results[1].Count)
	}
}

func TestGetUnchoppedExcludesMixedCommands(t *testing.T) {
	setupTestDB(t)

	// "git clone" sometimes compresses, sometimes not - should be excluded
	// (both share the same first-two-word key "git clone")
	if err := Track("git clone https://repo-a.git", 100, 50); err != nil {
		t.Fatal(err)
	}
	if err := Track("git clone https://repo-b.git", 100, 100); err != nil {
		t.Fatal(err)
	}

	// "env" never compresses - should appear
	if err := Track("env", 30, 30); err != nil {
		t.Fatal(err)
	}

	results, err := GetUnchopped()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 unchopped command, got %d", len(results))
	}
	if results[0].Command != "env" {
		t.Errorf("expected 'env', got %q", results[0].Command)
	}
}

func TestGetUnchoppedEmpty(t *testing.T) {
	setupTestDB(t)

	// Only compressed commands
	if err := Track("git status", 100, 20); err != nil {
		t.Fatal(err)
	}

	results, err := GetUnchopped()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 unchopped, got %d", len(results))
	}
}

func TestFormatUnchopped(t *testing.T) {
	summaries := []UnchoppedSummary{
		{Command: "ls -la", Count: 47, TotalTokens: 1234},
		{Command: "whoami", Count: 12, TotalTokens: 48},
	}
	out := FormatUnchopped(summaries, nil, nil, false)
	if !strings.Contains(out, "ls -la") {
		t.Errorf("missing command in output: %s", out)
	}
	if !strings.Contains(out, "47") {
		t.Errorf("missing count in output: %s", out)
	}
	if !strings.Contains(out, "1,234") {
		t.Errorf("missing token count in output: %s", out)
	}
	if !strings.Contains(out, "2 command(s)") && !strings.Contains(out, "command(s)") {
		t.Errorf("missing summary in output: %s", out)
	}
}

func TestFormatUnchoppedEmpty(t *testing.T) {
	out := FormatUnchopped(nil, nil, nil, false)
	if !strings.Contains(out, "all commands are being chopped") {
		t.Errorf("unexpected empty output: %s", out)
	}
}

func TestFormatUnchoppedWithSkipped(t *testing.T) {
	summaries := []UnchoppedSummary{
		{Command: "make tidy", Count: 9, TotalTokens: 128},
	}
	skipped := []string{"git push", "git tag"}
	out := FormatUnchopped(summaries, skipped, nil, false)
	if !strings.Contains(out, "make tidy") {
		t.Errorf("expected active command in output: %s", out)
	}
	if !strings.Contains(out, "skipped (no filter needed)") {
		t.Errorf("expected skipped section in output: %s", out)
	}
	if !strings.Contains(out, "git push") || !strings.Contains(out, "git tag") {
		t.Errorf("expected skipped commands in output: %s", out)
	}
}

func TestSkipAndUnskipUnchopped(t *testing.T) {
	setupTestDB(t)

	if err := Track("git push", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err := Track("make tidy", 50, 50); err != nil {
		t.Fatal(err)
	}

	// Both should appear before skipping
	results, err := GetUnchopped()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 before skip, got %d", len(results))
	}

	// Skip "git push"
	if err := SkipUnchopped("git push"); err != nil {
		t.Fatalf("SkipUnchopped failed: %v", err)
	}

	// Now only "make tidy" should appear
	results, err = GetUnchopped()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 after skip, got %d", len(results))
	}
	if results[0].Command != "make tidy" {
		t.Errorf("expected 'make tidy', got %q", results[0].Command)
	}

	// Skip list should contain "git push"
	skipped, err := GetSkippedCommands()
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped[0] != "git push" {
		t.Errorf("expected ['git push'] in skip list, got %v", skipped)
	}

	// Unskip restores it
	if err := UnskipUnchopped("git push"); err != nil {
		t.Fatalf("UnskipUnchopped failed: %v", err)
	}
	results, err = GetUnchopped()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 after unskip, got %d", len(results))
	}
}

func TestDeleteCommand(t *testing.T) {
	setupTestDB(t)

	if err := Track("git loh", 50, 50); err != nil {
		t.Fatal(err)
	}
	if err := Track("git loh origin", 30, 30); err != nil {
		t.Fatal(err)
	}
	if err := Track("git status", 100, 20); err != nil {
		t.Fatal(err)
	}

	if err := DeleteCommand("git loh"); err != nil {
		t.Fatalf("DeleteCommand failed: %v", err)
	}

	records, err := GetHistory(100)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		if strings.HasPrefix(r.Command, "git loh") {
			t.Errorf("expected git loh to be deleted, found: %q", r.Command)
		}
	}
	if len(records) != 1 || records[0].Command != "git status" {
		t.Errorf("expected only git status to remain, got %v", records)
	}
}

func TestFormatHistoryEmpty(t *testing.T) {
	out := FormatHistory(nil)
	if out != "no commands tracked yet" {
		t.Errorf("unexpected empty history: %s", out)
	}
}

func TestGetCommandSummary(t *testing.T) {
	setupTestDB(t)

	// insert some commands
	err := Track("npm test", 1000, 100)
	if err != nil {
		t.Fatalf("Failed to track npm test: %v", err)
	}
	err = Track("npm install", 500, 100)
	if err != nil {
		t.Fatalf("Failed to track npm install: %v", err)
	}
	err = Track("git status", 100, 100)
	if err != nil {
		t.Fatalf("Failed to track git status: %v", err)
	}

	summaries, err := GetCommandSummary()
	if err != nil {
		t.Fatalf("GetCommandSummary failed: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected 2 base commands, got %d", len(summaries))
	}

	// npm first (more savings)
	if summaries[0].BaseCommand != "npm" {
		t.Errorf("expected top command to be npm, got %s", summaries[0].BaseCommand)
	}
	if summaries[0].Count != 2 {
		t.Errorf("expected 2 calls for npm, got %d", summaries[0].Count)
	}
	if summaries[0].SavedTokens != 1300 {
		t.Errorf("expected 1300 saved tokens for npm, got %d", summaries[0].SavedTokens)
	}
	if summaries[0].ZeroCount != 0 {
		t.Errorf("expected 0 zero count for npm, got %d", summaries[0].ZeroCount)
	}

	if summaries[1].BaseCommand != "git" {
		t.Errorf("expected second command to be git, got %s", summaries[1].BaseCommand)
	}
	if summaries[1].Count != 1 {
		t.Errorf("expected 1 call for git, got %d", summaries[1].Count)
	}
	if summaries[1].SavedTokens != 0 {
		t.Errorf("expected 0 saved tokens for git, got %d", summaries[1].SavedTokens)
	}
	if summaries[1].ZeroCount != 1 {
		t.Errorf("expected 1 zero count for git, got %d", summaries[1].ZeroCount)
	}
}

func TestFormatSummary(t *testing.T) {
	summaries := []CommandSummary{
		{
			BaseCommand: "npm",
			Count:       2,
			SavedTokens: 1300,
			SavingsPct:  86.6,
			ZeroCount:   0,
		},
		{
			BaseCommand: "git",
			Count:       1,
			SavedTokens: 0,
			SavingsPct:  0,
			ZeroCount:   1,
		},
	}

	out := FormatSummary(summaries)
	if !strings.Contains(out, "npm") {
		t.Errorf("expected output to contain npm: %s", out)
	}
	if !strings.Contains(out, "1,300") {
		t.Errorf("expected output to contain 1,300: %s", out)
	}
	if !strings.Contains(out, "git") {
		t.Errorf("expected output to contain git: %s", out)
	}
	if !strings.Contains(out, "(1 calls at 0%)") {
		t.Errorf("expected output to contain '(1 calls at 0%%)': %s", out)
	}

	outEmpty := FormatSummary(nil)
	if outEmpty != "no commands tracked yet" {
		t.Errorf("expected empty message, got %s", outEmpty)
	}
}

func TestTrackingSkip(t *testing.T) {
	setupTestDB(t)

	err := AddTrackingSkip("docker ps")
	if err != nil {
		t.Fatalf("AddTrackingSkip failed: %v", err)
	}

	var skipped []string
	rows, err := db.Query(`SELECT command FROM tracking_skip ORDER BY command`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		skipped = append(skipped, cmd)
	}

	if len(skipped) != 1 || skipped[0] != "docker ps" {
		t.Errorf("expected [docker ps], got %v", skipped)
	}

	err = RemoveTrackingSkip("docker ps")
	if err != nil {
		t.Fatalf("RemoveTrackingSkip failed: %v", err)
	}

	skipped = nil
	rows, err = db.Query(`SELECT command FROM tracking_skip ORDER BY command`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		skipped = append(skipped, cmd)
	}

	if len(skipped) != 0 {
		t.Errorf("expected empty skipped list, got %v", skipped)
	}
}
