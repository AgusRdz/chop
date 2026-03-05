package filters

import (
	"strings"
	"testing"
)

func TestFilterMavenBuildSuccess(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[INFO]
[INFO] --- maven-resources-plugin:3.3.1:resources (default-resources) @ myapp ---
[INFO] Copying 1 resource from src/main/resources to target/classes
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:compile (default-compile) @ myapp ---
[INFO] Downloading from central: https://repo.maven.apache.org/maven2/org/apache/maven/shared/maven-shared-utils/3.3.4/maven-shared-utils-3.3.4.pom
[INFO] Downloaded from central: https://repo.maven.apache.org/maven2/org/apache/maven/shared/maven-shared-utils/3.3.4/maven-shared-utils-3.3.4.pom (4.5 kB at 120 kB/s)
[INFO] Downloading from central: https://repo.maven.apache.org/maven2/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar
[INFO] Downloaded from central: https://repo.maven.apache.org/maven2/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar (587 kB at 2.1 MB/s)
[INFO] Downloading from central: https://repo.maven.apache.org/maven2/org/ow2/asm/asm/9.5/asm-9.5.jar
[INFO] Downloaded from central: https://repo.maven.apache.org/maven2/org/ow2/asm/asm/9.5/asm-9.5.jar (123 kB at 1.8 MB/s)
[INFO] Nothing to compile - all classes are up to date
[INFO]
[INFO] --- maven-resources-plugin:3.3.1:testResources (default-testResources) @ myapp ---
[INFO] skip non existing resourceDirectory /home/user/myapp/src/test/resources
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:testCompile (default-testCompile) @ myapp ---
[INFO] Nothing to compile - all classes are up to date
[INFO]
[INFO] --- maven-jar-plugin:3.3.0:jar (default-jar) @ myapp ---
[INFO] Building jar: /home/user/myapp/target/myapp-1.0-SNAPSHOT.jar
[INFO]
[INFO] --- maven-install-plugin:3.1.1:install (default-install) @ myapp ---
[INFO] Installing /home/user/myapp/pom.xml to /home/user/.m2/repository/com/example/myapp/1.0-SNAPSHOT/myapp-1.0-SNAPSHOT.pom
[INFO] Installing /home/user/myapp/target/myapp-1.0-SNAPSHOT.jar to /home/user/.m2/repository/com/example/myapp/1.0-SNAPSHOT/myapp-1.0-SNAPSHOT.jar
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  5.432 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------`

	got, err := filterMavenBuild(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "BUILD SUCCESS") {
		t.Errorf("expected BUILD SUCCESS, got:\n%s", got)
	}
	if !strings.Contains(got, "5.432") {
		t.Errorf("expected elapsed time, got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 80.0 {
		t.Errorf("expected >=80%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterMavenBuildFailure(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:compile (default-compile) @ myapp ---
[ERROR] /home/user/myapp/src/main/java/com/example/App.java:[15,20] cannot find symbol
[ERROR]   symbol:   variable nonExistent
[ERROR]   location: class com.example.App
[ERROR] /home/user/myapp/src/main/java/com/example/Service.java:[42,10] incompatible types
[INFO] ------------------------------------------------------------------------
[INFO] BUILD FAILURE
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  2.100 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------
[ERROR] Failed to execute goal org.apache.maven.plugins:maven-compiler-plugin:3.11.0:compile (default-compile) on project myapp: Compilation failure: Compilation failure:
[ERROR] /home/user/myapp/src/main/java/com/example/App.java:[15,20] cannot find symbol
[ERROR]   symbol:   variable nonExistent
[ERROR]   location: class com.example.App`

	got, err := filterMavenBuild(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "BUILD FAILURE") {
		t.Errorf("expected BUILD FAILURE, got:\n%s", got)
	}
	if !strings.Contains(got, "cannot find symbol") {
		t.Errorf("expected error details preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "incompatible types") {
		t.Errorf("expected second error preserved, got:\n%s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestFilterMavenBuildWithWarnings(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[WARNING] Using platform encoding (UTF-8 actually) to copy filtered resources
[WARNING] Some problems were encountered while building the effective model
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:compile (default-compile) @ myapp ---
[INFO] Nothing to compile - all classes are up to date
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  1.200 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------`

	got, err := filterMavenBuild(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "BUILD SUCCESS") {
		t.Errorf("expected BUILD SUCCESS, got:\n%s", got)
	}
	if !strings.Contains(got, "2 warning(s)") {
		t.Errorf("expected warning count, got:\n%s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestFilterMavenTestAllPassing(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:compile (default-compile) @ myapp ---
[INFO] Nothing to compile - all classes are up to date
[INFO]
[INFO] --- maven-compiler-plugin:3.11.0:testCompile (default-testCompile) @ myapp ---
[INFO] Nothing to compile - all classes are up to date
[INFO]
[INFO] --- maven-surefire-plugin:3.1.2:test (default-test) @ myapp ---
[INFO] Downloading from central: https://repo.maven.apache.org/maven2/org/apache/maven/surefire/surefire-junit-platform/3.1.2/surefire-junit-platform-3.1.2.jar
[INFO] Downloaded from central: https://repo.maven.apache.org/maven2/org/apache/maven/surefire/surefire-junit-platform/3.1.2/surefire-junit-platform-3.1.2.jar (27 kB at 1.2 MB/s)
[INFO] Using auto detected provider org.apache.maven.surefire.junitplatform.JUnitPlatformProvider
[INFO]
[INFO] -------------------------------------------------------
[INFO]  T E S T S
[INFO] -------------------------------------------------------
[INFO] Running com.example.AppTest
[INFO] Tests run: 12, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 0.532 s -- in com.example.AppTest
[INFO] Running com.example.ServiceTest
[INFO] Tests run: 8, Failures: 0, Errors: 0, Skipped: 1, Time elapsed: 0.211 s -- in com.example.ServiceTest
[INFO] Running com.example.UtilTest
[INFO] Tests run: 5, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 0.089 s -- in com.example.UtilTest
[INFO]
[INFO] Results:
[INFO]
[INFO] Tests run: 25, Failures: 0, Errors: 0, Skipped: 1
[INFO]
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  8.732 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------`

	got, err := filterMavenTest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "all") && !strings.Contains(got, "passed") {
		t.Errorf("expected 'all N tests passed', got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 75.0 {
		t.Errorf("expected >=75%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterMavenTestWithFailures(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[INFO]
[INFO] --- maven-surefire-plugin:3.1.2:test (default-test) @ myapp ---
[INFO]
[INFO] -------------------------------------------------------
[INFO]  T E S T S
[INFO] -------------------------------------------------------
[INFO] Running com.example.AppTest
[ERROR] Tests run: 5, Failures: 2, Errors: 0, Skipped: 0, Time elapsed: 0.312 s <<< FAILURE! -- in com.example.AppTest
[ERROR]   testAddition(com.example.AppTest)  Time elapsed: 0.021 s  <<< FAILURE!
org.opentest4j.AssertionFailedError: expected: <4> but was: <5>
	at org.junit.jupiter.api.AssertEquals.assertEquals(AssertEquals.java:35)
[ERROR]   testSubtraction(com.example.AppTest)  Time elapsed: 0.003 s  <<< FAILURE!
org.opentest4j.AssertionFailedError: expected: <0> but was: <1>
	at org.junit.jupiter.api.AssertEquals.assertEquals(AssertEquals.java:35)
[INFO] Running com.example.ServiceTest
[INFO] Tests run: 8, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 0.201 s -- in com.example.ServiceTest
[INFO]
[INFO] Results:
[INFO]
[ERROR] Failures:
[ERROR]   AppTest.testAddition:15 expected: <4> but was: <5>
[ERROR]   AppTest.testSubtraction:22 expected: <0> but was: <1>
[INFO]
[ERROR] Tests run: 13, Failures: 2, Errors: 0, Skipped: 0
[INFO]
[INFO] ------------------------------------------------------------------------
[INFO] BUILD FAILURE
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  4.553 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------`

	got, err := filterMavenTest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "Failures: 2") {
		t.Errorf("expected failure count, got:\n%s", got)
	}
	if !strings.Contains(got, "Tests run: 13") {
		t.Errorf("expected test count, got:\n%s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestFilterMavenDepTree(t *testing.T) {
	raw := `[INFO] Scanning for projects...
[INFO]
[INFO] -----------------------< com.example:myapp >------------------------
[INFO] Building myapp 1.0-SNAPSHOT
[INFO] --------------------------------[ jar ]---------------------------------
[INFO]
[INFO] --- maven-dependency-plugin:3.6.0:tree (default-cli) @ myapp ---
[INFO] com.example:myapp:jar:1.0-SNAPSHOT
[INFO] +- org.springframework.boot:spring-boot-starter-web:jar:3.2.0:compile
[INFO] |  +- org.springframework.boot:spring-boot-starter:jar:3.2.0:compile
[INFO] |  |  +- org.springframework.boot:spring-boot:jar:3.2.0:compile
[INFO] |  |  +- org.springframework.boot:spring-boot-autoconfigure:jar:3.2.0:compile
[INFO] |  |  +- org.springframework.boot:spring-boot-starter-logging:jar:3.2.0:compile
[INFO] |  |  |  +- ch.qos.logback:logback-classic:jar:1.4.14:compile
[INFO] |  |  |  |  +- ch.qos.logback:logback-core:jar:1.4.14:compile
[INFO] |  |  |  +- org.apache.logging.log4j:log4j-to-slf4j:jar:2.21.1:compile
[INFO] |  |  |  |  +- org.apache.logging.log4j:log4j-api:jar:2.21.1:compile
[INFO] |  |  |  +- org.slf4j:jul-to-slf4j:jar:2.0.9:compile
[INFO] |  |  +- jakarta.annotation:jakarta.annotation-api:jar:2.1.1:compile
[INFO] |  |  +- org.yaml:snakeyaml:jar:2.2:compile
[INFO] |  +- org.springframework.boot:spring-boot-starter-json:jar:3.2.0:compile
[INFO] |  |  +- com.fasterxml.jackson.core:jackson-databind:jar:2.15.3:compile
[INFO] |  |  |  +- com.fasterxml.jackson.core:jackson-annotations:jar:2.15.3:compile
[INFO] |  |  |  +- com.fasterxml.jackson.core:jackson-core:jar:2.15.3:compile
[INFO] |  +- org.springframework:spring-web:jar:6.1.1:compile
[INFO] |  +- org.springframework:spring-webmvc:jar:6.1.1:compile
[INFO] +- org.postgresql:postgresql:jar:42.7.1:runtime
[INFO] +- org.projectlombok:lombok:jar:1.18.30:compile
[INFO] \- junit:junit:jar:4.13.2:test
[INFO]    \- org.hamcrest:hamcrest-core:jar:1.3:test
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  1.234 s
[INFO] Finished at: 2026-03-05T10:30:00Z
[INFO] ------------------------------------------------------------------------`

	got, err := filterMavenDepTree(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "spring-boot-starter-web") {
		t.Errorf("expected direct dep spring-boot-starter-web, got:\n%s", got)
	}
	if !strings.Contains(got, "postgresql") {
		t.Errorf("expected direct dep postgresql, got:\n%s", got)
	}
	if !strings.Contains(got, "4 direct") {
		t.Errorf("expected 4 direct deps, got:\n%s", got)
	}
	if !strings.Contains(got, "transitive") {
		t.Errorf("expected transitive count, got:\n%s", got)
	}

	rawTokens := len(strings.Fields(raw))
	filteredTokens := len(strings.Fields(got))
	savings := 100.0 - float64(filteredTokens)/float64(rawTokens)*100.0
	if savings < 60.0 {
		t.Errorf("expected >=60%% token savings, got %.1f%% (raw=%d, filtered=%d)", savings, rawTokens, filteredTokens)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestFilterMavenBuildEmpty(t *testing.T) {
	got, err := filterMavenBuild("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

func TestFilterMavenTestEmpty(t *testing.T) {
	got, err := filterMavenTest("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}
