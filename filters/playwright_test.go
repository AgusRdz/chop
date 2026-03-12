package filters

import (
	"strings"
	"testing"
)

func TestPlaywrightAllPassed(t *testing.T) {
	raw := `Running 5 tests using 2 workers

  ✓  1 [chromium] › tests/home.spec.ts:3:1 › has title (1.2s)
  ✓  2 [chromium] › tests/home.spec.ts:8:1 › get started link (1.3s)
  ✓  3 [firefox] › tests/home.spec.ts:3:1 › has title (1.4s)
  ✓  4 [firefox] › tests/home.spec.ts:8:1 › get started link (1.5s)
  ✓  5 [webkit] › tests/home.spec.ts:3:1 › has title (0.9s)

  5 passed (2.3s)
`
	got, err := filterPlaywright(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "all 5 tests passed" {
		t.Errorf("expected 'all 5 tests passed', got: %q", got)
	}
	rawTokens := countTokens(raw)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 70.0 {
		t.Errorf("expected >=70%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
}

func TestPlaywrightWithFailures(t *testing.T) {
	raw := `Running 3 tests using 2 workers

  ✓  1 [chromium] › tests/home.spec.ts:3:1 › has title (1.2s)
  ✗  2 [chromium] › tests/auth.spec.ts:10:1 › login should work (500ms)
  ✓  3 [firefox] › tests/home.spec.ts:3:1 › has title (1.4s)

  1) [chromium] › tests/auth.spec.ts:10:1 › login should work

    Error: expect(received).toBe(expected)

    Expected: true
    Received: false

      10 |   expect(result).toBe(true);
      11 | });

    at tests/auth.spec.ts:10:20

  3 tests (1 failed) - 4.5s
`
	got, err := filterPlaywright(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "2 passed") {
		t.Errorf("expected '2 passed' in output, got: %s", got)
	}
	if !strings.Contains(got, "1 failed") {
		t.Errorf("expected '1 failed' in output, got: %s", got)
	}
	if !strings.Contains(got, "login should work") {
		t.Errorf("expected failure detail in output, got: %s", got)
	}
	rawTokens := countTokens(raw)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 40.0 {
		t.Errorf("expected >=40%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestPlaywrightWithRetriesCountsOnce(t *testing.T) {
	raw := `Running 1 test using 1 worker

  ×  1 [chromium] › tests/flaky.spec.ts:3:1 › flaky test – 1/3 (400ms)
  ×  1 [chromium] › tests/flaky.spec.ts:3:1 › flaky test – 2/3 (350ms)
  ✗  1 [chromium] › tests/flaky.spec.ts:3:1 › flaky test – 3/3 (420ms)

  1) [chromium] › tests/flaky.spec.ts:3:1 › flaky test

    Error: timeout exceeded

  1 test (1 failed) - 5.2s
`
	got, err := filterPlaywright(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "0 passed, 1 failed") {
		t.Errorf("expected '0 passed, 1 failed', got: %s", got)
	}
	t.Logf("output: %s", got)
}

func TestPlaywrightNonPlaywrightOutput(t *testing.T) {
	raw := "some random output that is not playwright\n"
	got, err := filterPlaywright(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("expected raw output unchanged, got: %q", got)
	}
}

func TestPlaywrightDotReporterFallback(t *testing.T) {
	raw := `......F..

  1) Error: expect(received).toBe(expected)

  9 tests (1 failed) - 1.2s
`
	got, err := filterPlaywright(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "8 passed") || !strings.Contains(got, "1 failed") {
		t.Errorf("expected '8 passed, 1 failed', got: %s", got)
	}
	t.Logf("output: %s", got)
}
