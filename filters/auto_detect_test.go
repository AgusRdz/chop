package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestAutoDetect_Empty(t *testing.T) {
	got, err := AutoDetect("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestAutoDetect_ShortPassthrough(t *testing.T) {
	input := "line1\nline2\nline3\n"
	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("short input should pass through unchanged")
	}
}

func TestAutoDetect_JSON(t *testing.T) {
	// Build a large JSON array to exceed the short threshold
	var items []string
	for i := 0; i < 30; i++ {
		items = append(items, fmt.Sprintf(`{"id": %d, "name": "item_%d", "status": "active", "score": %d, "category": "test", "extra": "value"}`, i, i, i*10))
	}
	input := "[\n" + strings.Join(items, ",\n") + "\n]"

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be compressed by compressJSON (structure summary, not full values)
	if strings.Contains(got, `"item_29"`) {
		t.Error("JSON should be compressed, but last item value still present")
	}
	if got == input {
		t.Error("JSON input should have been compressed")
	}
}

func TestAutoDetect_CSV(t *testing.T) {
	var b strings.Builder
	b.WriteString("Name,Age,City,Country\n")
	for i := 0; i < 25; i++ {
		b.WriteString(fmt.Sprintf("Person%d,%d,City%d,Country%d\n", i, 20+i, i, i))
	}
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "Name,Age,City,Country") {
		t.Error("CSV header should be preserved")
	}
	if !strings.Contains(got, "... and 20 more rows") {
		t.Errorf("should show remaining row count, got:\n%s", got)
	}
	if !strings.Contains(got, "(4 columns)") {
		t.Error("should show column count")
	}

	// Verify savings >= 50%
	rawTokens := len(strings.Fields(input))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 50 {
		t.Errorf("CSV savings %.1f%% < 50%%", savings)
	}
}

func TestAutoDetect_Table(t *testing.T) {
	var b strings.Builder
	b.WriteString("+----------+-------+--------+\n")
	b.WriteString("| Name     | Score | Status |\n")
	b.WriteString("+----------+-------+--------+\n")
	for i := 0; i < 25; i++ {
		b.WriteString(fmt.Sprintf("| User%-4d | %5d | active |\n", i, i*10))
	}
	b.WriteString("+----------+-------+--------+\n")
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Header should be present
	if !strings.Contains(got, "Name") && !strings.Contains(got, "Score") {
		t.Error("table header should be preserved")
	}
	// Should show truncation
	if !strings.Contains(got, "... and") {
		t.Errorf("should truncate table rows, got:\n%s", got)
	}
}

func TestAutoDetect_LogLike(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString(fmt.Sprintf("2026-03-05T10:%02d:00 INFO Starting process %d\n", i, i%5))
	}
	for i := 0; i < 40; i++ {
		b.WriteString(fmt.Sprintf("2026-03-05T10:%02d:00 DEBUG Detailed debug info line %d\n", i, i%5))
	}
	for i := 0; i < 20; i++ {
		b.WriteString("2026-03-05T11:00:00 INFO Repeated message\n")
	}
	b.WriteString("2026-03-05T11:01:00 ERROR Something failed\n")
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should deduplicate repeated messages
	if strings.Count(got, "Repeated message") > 1 {
		t.Error("repeated messages should be deduplicated")
	}
	// Should contain (xN) count
	if !strings.Contains(got, "(x") {
		t.Errorf("should show repeat count, got:\n%s", got)
	}
	// ERROR should always be kept
	if !strings.Contains(got, "ERROR") {
		t.Error("ERROR lines should always be kept")
	}

	rawTokens := len(strings.Fields(input))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 60 {
		t.Errorf("log savings %.1f%% < 60%%", savings)
	}
}

func TestAutoDetect_XML(t *testing.T) {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<root>\n")
	for i := 0; i < 30; i++ {
		b.WriteString(fmt.Sprintf("  <item id=\"%d\"><name>Item %d</name><value>%d</value></item>\n", i, i, i*100))
	}
	b.WriteString("</root>\n")
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(got, "XML response") {
		t.Errorf("large XML should be summarized, got: %s", got)
	}
	if !strings.Contains(got, "bytes") {
		t.Error("should show byte count")
	}
}

func TestAutoDetect_HTML(t *testing.T) {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head><title>Test</title></head>\n<body>\n")
	for i := 0; i < 30; i++ {
		b.WriteString(fmt.Sprintf("<p>Paragraph %d with some content here</p>\n", i))
	}
	b.WriteString("</body>\n</html>\n")
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(got, "HTML response") {
		t.Errorf("large HTML should be summarized, got: %s", got)
	}
}

func TestAutoDetect_LongPlainText(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString(fmt.Sprintf("This is line number %d of some plain text output from a command\n", i))
	}
	input := b.String()

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "lines hidden") {
		t.Errorf("long text should be truncated, got:\n%s", got)
	}
	// First line should be present
	if !strings.Contains(got, "line number 0") {
		t.Error("first lines should be preserved")
	}
	// Last line should be present
	if !strings.Contains(got, "line number 99") {
		t.Error("last lines should be preserved")
	}

	rawTokens := len(strings.Fields(input))
	gotTokens := len(strings.Fields(got))
	savings := 100.0 - float64(gotTokens)/float64(rawTokens)*100.0
	if savings < 50 {
		t.Errorf("plain text savings %.1f%% < 50%%", savings)
	}
}

func TestAutoDetect_SmallXMLPassthrough(t *testing.T) {
	input := `<?xml version="1.0"?>
<config>
  <setting name="debug">true</setting>
  <setting name="timeout">30</setting>
</config>`

	got, err := AutoDetect(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Small XML (<20 lines, <500 chars) should pass through
	if got != input {
		t.Errorf("small XML should pass through unchanged")
	}
}
