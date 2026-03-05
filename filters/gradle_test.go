package filters

import (
	"strings"
	"testing"
)

func TestFilterGradleBuildSuccess(t *testing.T) {
	raw := `Starting a Gradle Daemon (subsequent builds will be faster)

> Task :compileJava
> Task :processResources NO-SOURCE
> Task :classes
> Task :jar
> Task :compileTestJava
> Task :processTestResources NO-SOURCE
> Task :testClasses
> Task :test
> Task :check
> Task :build

BUILD SUCCESSFUL in 12s
7 actionable tasks: 7 executed

Deprecated Gradle features were used in this build, making it incompatible with Gradle 9.0.
To honour the JVM settings for this build a new JVM was forked.`

	got, err := filterGradleBuild(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "BUILD SUCCESSFUL") {
		t.Errorf("expected BUILD SUCCESSFUL, got:\n%s", got)
	}
	if !strings.Contains(got, "12s") {
		t.Errorf("expected elapsed time, got:\n%s", got)
	}
	if !strings.Contains(got, "7 tasks") {
		t.Errorf("expected task count, got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 70.0 {
		t.Errorf("expected >=70%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterGradleBuildFailure(t *testing.T) {
	raw := `> Task :compileJava
> Task :processResources NO-SOURCE
> Task :classes
> Task :compileTestJava FAILED

FAILURE: Build failed with an exception.

* What went wrong:
Execution failed for task ':compileTestJava'.
> Compilation failed; see the compiler error output for details.

* Try:
> Run with --stacktrace option to get the stack trace.
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Get more help at https://help.gradle.org

BUILD FAILED in 3s
4 actionable tasks: 2 executed, 2 up-to-date`

	got, err := filterGradleBuild(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "BUILD FAILED") {
		t.Errorf("expected BUILD FAILED, got:\n%s", got)
	}
	if !strings.Contains(got, "Compilation failed") {
		t.Errorf("expected error details preserved, got:\n%s", got)
	}
	if !strings.Contains(got, ":compileTestJava") {
		t.Errorf("expected failed task name, got:\n%s", got)
	}
	// Should NOT contain "Try:" section
	if strings.Contains(got, "Try:") || strings.Contains(got, "--stacktrace") {
		t.Errorf("should strip Try section, got:\n%s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestFilterGradleTestAllPassing(t *testing.T) {
	raw := `> Task :compileJava UP-TO-DATE
> Task :processResources NO-SOURCE
> Task :classes UP-TO-DATE
> Task :compileTestJava UP-TO-DATE
> Task :processTestResources NO-SOURCE
> Task :testClasses UP-TO-DATE
> Task :test

42 tests completed, 0 failed, 2 skipped

BUILD SUCCESSFUL in 8s
4 actionable tasks: 1 executed, 3 up-to-date

Deprecated Gradle features were used in this build, making it incompatible with Gradle 9.0.`

	got, err := filterGradleTest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "all 42 tests passed") {
		t.Errorf("expected 'all 42 tests passed', got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 70.0 {
		t.Errorf("expected >=70%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterGradleTestWithFailures(t *testing.T) {
	raw := `> Task :compileJava UP-TO-DATE
> Task :processResources NO-SOURCE
> Task :classes UP-TO-DATE
> Task :compileTestJava UP-TO-DATE
> Task :processTestResources NO-SOURCE
> Task :testClasses UP-TO-DATE
> Task :test

com.example.AppTest > testAddition FAILED
    org.opentest4j.AssertionFailedError: expected: <4> but was: <5>

com.example.AppTest > testSubtraction FAILED
    org.opentest4j.AssertionFailedError: expected: <0> but was: <1>

20 tests completed, 2 failed

> Task :test FAILED

FAILURE: Build failed with an exception.

* What went wrong:
Execution failed for task ':test'.

* Try:
> Run with --stacktrace option to get the stack trace.

* Get more help at https://help.gradle.org

BUILD FAILED in 5s
4 actionable tasks: 1 executed, 3 up-to-date`

	got, err := filterGradleTest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "testAddition") {
		t.Errorf("expected failed test name, got:\n%s", got)
	}
	if !strings.Contains(got, "testSubtraction") {
		t.Errorf("expected second failed test, got:\n%s", got)
	}
	if !strings.Contains(got, "2 failed") {
		t.Errorf("expected failure count, got:\n%s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestFilterGradleDeps(t *testing.T) {
	raw := `> Task :dependencies

------------------------------------------------------------
Project ':app' - Main application
------------------------------------------------------------

implementation - Implementation dependencies for the 'main' feature.
+--- org.springframework.boot:spring-boot-starter-web:3.2.0
|    +--- org.springframework.boot:spring-boot-starter:3.2.0
|    |    +--- org.springframework.boot:spring-boot:3.2.0
|    |    +--- org.springframework.boot:spring-boot-autoconfigure:3.2.0
|    |    +--- org.springframework.boot:spring-boot-starter-logging:3.2.0
|    |    |    +--- ch.qos.logback:logback-classic:1.4.14
|    |    |    +--- org.apache.logging.log4j:log4j-to-slf4j:2.21.1
|    |    |    \--- org.slf4j:jul-to-slf4j:2.0.9
|    |    +--- jakarta.annotation:jakarta.annotation-api:2.1.1
|    |    \--- org.yaml:snakeyaml:2.2
|    +--- org.springframework.boot:spring-boot-starter-json:3.2.0
|    +--- org.springframework:spring-web:6.1.1
|    \--- org.springframework:spring-webmvc:6.1.1
+--- org.postgresql:postgresql:42.7.1
\--- com.google.guava:guava:32.1.3-jre
     +--- com.google.guava:failureaccess:1.0.1
     +--- com.google.guava:listenablefuture:9999.0-empty-to-avoid-conflict-with-guava
     \--- com.google.code.findbugs:jsr305:3.0.2

BUILD SUCCESSFUL in 1s
1 actionable task: 1 executed`

	got, err := filterGradleDeps(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "spring-boot-starter-web") {
		t.Errorf("expected direct dep, got:\n%s", got)
	}
	if !strings.Contains(got, "postgresql") {
		t.Errorf("expected direct dep postgresql, got:\n%s", got)
	}
	if !strings.Contains(got, "guava") {
		t.Errorf("expected direct dep guava, got:\n%s", got)
	}
	if !strings.Contains(got, "3 direct") {
		t.Errorf("expected 3 direct deps, got:\n%s", got)
	}
	if !strings.Contains(got, "transitive") {
		t.Errorf("expected transitive count, got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 50.0 {
		t.Errorf("expected >=50%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterGradleBuildEmpty(t *testing.T) {
	got, err := filterGradleBuild("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

func TestFilterGradleTestEmpty(t *testing.T) {
	got, err := filterGradleTest("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}
