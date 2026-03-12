package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestPatternFingerprint_UUID(t *testing.T) {
	line := "Processing request a1b2c3d4-e5f6-7890-abcd-ef1234567890 complete"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "a1b2c3d4") {
		t.Errorf("UUID should be replaced, got: %s", fp)
	}
	if !strings.Contains(fp, "<*>") {
		t.Errorf("should contain placeholder, got: %s", fp)
	}
	if !strings.Contains(fp, "Processing request") {
		t.Errorf("static text should be preserved, got: %s", fp)
	}
}

func TestPatternFingerprint_IPAddress(t *testing.T) {
	line := "Connection to 10.0.0.5:3306 established"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "10.0.0.5") {
		t.Errorf("IP should be replaced, got: %s", fp)
	}
	if !strings.Contains(fp, "Connection to") {
		t.Errorf("static text should be preserved, got: %s", fp)
	}
}

func TestPatternFingerprint_Duration(t *testing.T) {
	line := "Request completed in 45ms"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "45ms") {
		t.Errorf("duration should be replaced, got: %s", fp)
	}
}

func TestPatternFingerprint_KeyValue(t *testing.T) {
	line := "INFO Processing id=abc123 duration=45ms status=ok"
	fp := patternFingerprint(line)
	if !strings.Contains(fp, "id=") {
		t.Errorf("key= prefix should be preserved, got: %s", fp)
	}
	if strings.Contains(fp, "abc123") {
		t.Errorf("value should be replaced, got: %s", fp)
	}
}

func TestPatternFingerprint_Numbers(t *testing.T) {
	line := "Processed 42 items in batch 7"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "42") || strings.Contains(fp, " 7") {
		t.Errorf("numbers should be replaced, got: %s", fp)
	}
	if !strings.Contains(fp, "Processed") {
		t.Errorf("static text should be preserved, got: %s", fp)
	}
}

func TestPatternFingerprint_HexID(t *testing.T) {
	line := "Commit abc123def456 applied to branch"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "abc123def456") {
		t.Errorf("hex ID should be replaced, got: %s", fp)
	}
}

func TestPatternFingerprint_QuotedStrings(t *testing.T) {
	line := `Loaded module "auth-service" from "lib/auth"`
	fp := patternFingerprint(line)
	if strings.Contains(fp, "auth-service") {
		t.Errorf("quoted string should be replaced, got: %s", fp)
	}
}

func TestPatternFingerprint_ISOTimestamp(t *testing.T) {
	line := "Event at 2024-03-11T10:00:01.123Z processed"
	fp := patternFingerprint(line)
	if strings.Contains(fp, "2024-03-11T10:00:01") {
		t.Errorf("ISO timestamp should be replaced, got: %s", fp)
	}
}

func TestHasEnoughStaticTokens(t *testing.T) {
	tests := []struct {
		fp     string
		expect bool
	}{
		{"INFO Processing request id=<*> duration=<*>", true},
		{"<*> <*> <*>", false},
		{"<*>", false},
		{"ERROR Connection timeout to <*>", true},
		{"Processing complete", true},
	}
	for _, tt := range tests {
		got := hasEnoughStaticTokens(tt.fp)
		if got != tt.expect {
			t.Errorf("hasEnoughStaticTokens(%q) = %v, want %v", tt.fp, got, tt.expect)
		}
	}
}

func TestCompressLogPatterns_RepetitiveLogs(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:%02d:%02d INFO Processing request id=req%04d duration=%dms", i/60, i%60, i, 30+i))
	}
	lines = append(lines, "2024-03-11 11:00:00 ERROR Connection timeout to 10.0.0.5:3306")

	isImportant := func(line string) bool {
		return strings.Contains(strings.ToUpper(line), "ERROR")
	}

	result, ok := compressLogPatterns(lines, isImportant)
	if !ok {
		t.Fatal("should have applied pattern compression")
	}

	// Should have the pattern with count
	if !strings.Contains(result, "(x") {
		t.Errorf("should contain pattern count, got:\n%s", result)
	}

	// ERROR should be preserved
	if !strings.Contains(result, "ERROR") {
		t.Errorf("ERROR should be preserved, got:\n%s", result)
	}

	// Should be significantly compressed
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > 10 {
		t.Errorf("expected significant compression, got %d lines:\n%s", len(resultLines), result)
	}

	// Token savings should be substantial
	rawTokens := 0
	for _, l := range lines {
		rawTokens += len(strings.Fields(l))
	}
	gotTokens := len(strings.Fields(result))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 70 {
		t.Errorf("pattern compression savings %.1f%% < 70%%", savings)
	}
}

func TestCompressLogPatterns_TooFewLines(t *testing.T) {
	lines := []string{
		"2024-03-11 10:00:00 INFO Starting",
		"2024-03-11 10:00:01 INFO Ready",
	}
	isImportant := func(string) bool { return false }

	_, ok := compressLogPatterns(lines, isImportant)
	if ok {
		t.Error("should not apply pattern compression to very few lines")
	}
}

func TestCompressLogPatterns_NoRepeats(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:00:%02d INFO Unique message number %d with different structure each time", i, i))
	}
	isImportant := func(string) bool { return false }

	// If all lines share a pattern, it should group them.
	// If lines are truly unique in structure, it falls back.
	result, ok := compressLogPatterns(lines, isImportant)
	if ok {
		// If it did match (because structure IS similar), verify compression
		if len(strings.Split(result, "\n")) > len(lines) {
			t.Errorf("if patterns matched, output should not be larger than input")
		}
	}
	// Either way is fine - it should not crash
}

func TestCompressLogPatterns_MixedPatterns(t *testing.T) {
	var lines []string
	// Pattern A: request processing
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:00:%02d INFO Processing request id=req%04d duration=%dms", i, i, 30+i))
	}
	// Pattern B: health checks
	for i := 0; i < 15; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:01:%02d INFO Health check from 192.168.1.%d status=ok", i, i+1))
	}
	// Pattern C: single error
	lines = append(lines, "2024-03-11 10:02:00 ERROR Database connection pool exhausted")

	isImportant := func(line string) bool {
		return strings.Contains(strings.ToUpper(line), "ERROR")
	}

	result, ok := compressLogPatterns(lines, isImportant)
	if !ok {
		t.Fatal("should have applied pattern compression")
	}

	resultLines := strings.Split(result, "\n")
	// Should be compressed to ~3 groups (2 patterns + 1 unique error)
	if len(resultLines) > 5 {
		t.Errorf("expected ~3 pattern groups, got %d lines:\n%s", len(resultLines), result)
	}

	// ERROR must be present
	if !strings.Contains(result, "ERROR") {
		t.Error("ERROR line should be preserved")
	}

	t.Logf("Pattern compression result:\n%s", result)
}

func TestCompressLogPatterns_ErrorsAlwaysShown(t *testing.T) {
	var lines []string
	// Many normal lines
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:00:%02d INFO Normal operation %d", i%60, i))
	}
	// Errors scattered
	lines = append(lines, "2024-03-11 10:01:00 ERROR Disk space critical on /dev/sda1")
	lines = append(lines, "2024-03-11 10:01:01 WARN Memory usage high at 95%")

	isImportant := func(line string) bool {
		upper := strings.ToUpper(line)
		return strings.Contains(upper, "ERROR") || strings.Contains(upper, "WARN")
	}

	result, ok := compressLogPatterns(lines, isImportant)
	if !ok {
		t.Fatal("should have applied pattern compression")
	}

	if !strings.Contains(result, "ERROR") {
		t.Error("ERROR should always be shown")
	}
	if !strings.Contains(result, "WARN") {
		t.Error("WARN should always be shown")
	}
}

func TestCompressLogPatterns_SingleOccurrenceShowsOriginal(t *testing.T) {
	var lines []string
	// Repeated pattern
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:00:%02d INFO Request processed id=%d", i, i))
	}
	// Unique line
	lines = append(lines, "2024-03-11 10:01:00 INFO Server startup complete, listening on port 8080")

	isImportant := func(string) bool { return false }

	result, ok := compressLogPatterns(lines, isImportant)
	if !ok {
		t.Fatal("should have applied pattern compression")
	}

	// The unique line should appear unchanged (no placeholders)
	if !strings.Contains(result, "Server startup complete") {
		t.Errorf("unique line should be shown as-is, got:\n%s", result)
	}
}

func TestCompressLog_UsesPatternCompression(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 60; i++ {
		b.WriteString(fmt.Sprintf("2024-03-11 10:%02d:%02d INFO Processing request id=req%04d duration=%dms\n", i/60, i%60, i, 30+i))
	}
	b.WriteString("2024-03-11 11:00:00 ERROR Connection timeout\n")
	input := strings.TrimSpace(b.String())
	lines := strings.Split(input, "\n")

	result := compressLog(lines)

	// Should be significantly compressed via pattern matching
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > 10 {
		t.Errorf("compressLog should use pattern compression, got %d lines:\n%s", len(resultLines), result)
	}

	if !strings.Contains(result, "ERROR") {
		t.Error("ERROR should be preserved")
	}

	if !strings.Contains(result, "(x") {
		t.Errorf("should show pattern count, got:\n%s", result)
	}
}

func TestFilterTextLogs_UsesPatternCompression(t *testing.T) {
	var lines []string
	for i := 0; i < 60; i++ {
		lines = append(lines, fmt.Sprintf("2024-03-11 10:%02d:%02d INFO Request from 192.168.1.%d processed in %dms", i/60, i%60, i%254+1, 20+i))
	}
	lines = append(lines, "2024-03-11 11:00:00 ERROR Service unavailable")

	result := filterTextLogs(lines)

	resultLines := strings.Split(result, "\n")
	if len(resultLines) > 10 {
		t.Errorf("filterTextLogs should use pattern compression, got %d lines:\n%s", len(resultLines), result)
	}

	if !strings.Contains(result, "ERROR") {
		t.Error("ERROR should be preserved")
	}
}

func TestCompressLogPatterns_ConsistentFingerprints(t *testing.T) {
	// Same structure should produce same fingerprint
	line1 := "INFO Processing request id=abc123 duration=45ms"
	line2 := "INFO Processing request id=xyz789 duration=100ms"

	fp1 := patternFingerprint(line1)
	fp2 := patternFingerprint(line2)

	if fp1 != fp2 {
		t.Errorf("same structure should produce same fingerprint:\n  %s\n  %s", fp1, fp2)
	}
}

func TestCompressLogPatterns_DifferentStructures(t *testing.T) {
	line1 := "INFO Processing request id=abc123 duration=45ms"
	line2 := "INFO Health check completed successfully"

	fp1 := patternFingerprint(line1)
	fp2 := patternFingerprint(line2)

	if fp1 == fp2 {
		t.Errorf("different structures should produce different fingerprints:\n  %s\n  %s", fp1, fp2)
	}
}
