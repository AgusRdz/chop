package filters

import (
	"fmt"
	"strings"
	"testing"
)

func TestGitDiffMultiFile(t *testing.T) {
	raw := `diff --git a/src/app.ts b/src/app.ts
index 1234567..abcdefg 100644
--- a/src/app.ts
+++ b/src/app.ts
@@ -1,5 +1,8 @@
 import express from 'express';
+import cors from 'cors';
+import helmet from 'helmet';

 const app = express();
+app.use(cors());
+app.use(helmet());

-app.listen(3000);
+app.listen(process.env.PORT || 3000);
diff --git a/src/auth/login.ts b/src/auth/login.ts
index 2345678..bcdefgh 100644
--- a/src/auth/login.ts
+++ b/src/auth/login.ts
@@ -10,8 +10,6 @@
 export async function login(email: string, password: string) {
-  const user = await db.query('SELECT * FROM users WHERE email = ?', [email]);
-  if (!user) throw new Error('User not found');
-  const valid = await bcrypt.compare(password, user.password);
+  const user = await findUserByEmail(email);
+  const valid = await verifyPassword(password, user);
   if (!valid) throw new Error('Invalid credentials');
   return generateToken(user);
 }
diff --git a/package.json b/package.json
index 3456789..cdefghi 100644
--- a/package.json
+++ b/package.json
@@ -5,6 +5,8 @@
   "dependencies": {
     "express": "^4.18.0",
+    "cors": "^2.8.5",
+    "helmet": "^7.1.0",
     "bcrypt": "^5.1.0"
   }
 }
`

	got, err := filterGitDiff(raw)
	if err != nil {
		t.Fatal(err)
	}

	// All three file diffs and their actual hunk content must survive —
	// none of the context runs here are long enough to elide.
	if got != strings.TrimSpace(raw) {
		t.Errorf("expected full diff to pass through unchanged, got: %s", got)
	}
}

func TestGitDiffElidesLongContextRuns(t *testing.T) {
	var body strings.Builder
	body.WriteString("diff --git a/src/big.go b/src/big.go\n")
	body.WriteString("index 1234567..abcdefg 100644\n")
	body.WriteString("--- a/src/big.go\n")
	body.WriteString("+++ b/src/big.go\n")
	body.WriteString("@@ -1,40 +1,40 @@\n")
	body.WriteString(" package big\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&body, " unchanged line %d\n", i)
	}
	body.WriteString("-old line\n")
	body.WriteString("+new line\n")

	raw := body.String()

	got, err := filterGitDiff(raw)
	if err != nil {
		t.Fatal(err)
	}

	// Headers and the actual change must survive.
	if !strings.Contains(got, "diff --git a/src/big.go b/src/big.go") {
		t.Errorf("expected diff header to survive, got: %s", got)
	}
	if !strings.Contains(got, "-old line") || !strings.Contains(got, "+new line") {
		t.Errorf("expected changed lines to survive, got: %s", got)
	}

	// The long unchanged run must be elided, not silently dropped.
	if !strings.Contains(got, "unchanged lines elided") {
		t.Errorf("expected elision marker for long context run, got: %s", got)
	}

	// A few lines of context should remain on each side of the elision.
	if !strings.Contains(got, "unchanged line 0") {
		t.Errorf("expected leading context to survive, got: %s", got)
	}
	if !strings.Contains(got, "unchanged line 29") {
		t.Errorf("expected trailing context to survive, got: %s", got)
	}

	if len(got) >= len(raw) {
		t.Errorf("expected elided output to be shorter than raw: got %d, raw %d", len(got), len(raw))
	}
}

func TestGitDiffShortPassthrough(t *testing.T) {
	raw := `diff --git a/README.md b/README.md
index 1234567..abcdefg 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,3 @@
-# Old Title
+# New Title

 Some content.`

	got, err := filterGitDiff(raw)
	if err != nil {
		t.Fatal(err)
	}

	// Short diff (<10 lines) should pass through
	if got != strings.TrimSpace(raw) {
		t.Errorf("short diff should pass through, got: %s", got)
	}
}

func TestGitDiffEmpty(t *testing.T) {
	got, err := filterGitDiff("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty output for empty input, got: %s", got)
	}
}
