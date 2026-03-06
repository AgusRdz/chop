package filters

import (
	"strings"
	"testing"
)

func TestFilterPipInstall(t *testing.T) {
	raw := `Collecting flask
  Downloading flask-3.0.0-py3-none-any.whl (101 kB)
Collecting werkzeug>=3.0.0
  Downloading werkzeug-3.0.1-py3-none-any.whl (226 kB)
Collecting jinja2>=3.1.2
  Using cached jinja2-3.1.2-py3-none-any.whl (133 kB)
Installing collected packages: werkzeug, jinja2, flask
Successfully installed flask-3.0.0 jinja2-3.1.2 werkzeug-3.0.1`

	got, err := filterPipInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
	if !strings.Contains(got, "installed 3 packages") {
		t.Errorf("expected package count, got: %s", got)
	}
}

func TestFilterPipList(t *testing.T) {
	raw := "Package    Version\n---------- -------\nflask      3.0.0\njinja2     3.1.2\nwerkzeug   3.0.1\n"

	got, err := filterPipList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "3 packages installed") {
		t.Errorf("expected count, got: %s", got)
	}
}

func TestFilterPipInstall_Empty(t *testing.T) {
	got, err := filterPipInstall("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterPipInstall_LargeWithUninstall(t *testing.T) {
	raw := `Collecting django==5.0.2
  Downloading Django-5.0.2-py3-none-any.whl (8.2 MB)
     ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 8.2/8.2 MB 12.4 MB/s eta 0:00:00
Collecting asgiref>=3.7.0 (from django==5.0.2)
  Using cached asgiref-3.7.2-py3-none-any.whl (24 kB)
Collecting sqlparse>=0.3.1 (from django==5.0.2)
  Using cached sqlparse-0.4.4-py3-none-any.whl (41 kB)
Collecting tzdata (from django==5.0.2)
  Downloading tzdata-2024.1-py2.py3-none-any.whl (345 kB)
     ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 345.4/345.4 kB 8.7 MB/s eta 0:00:00
Requirement already satisfied: setuptools in /usr/lib/python3/dist-packages (from django==5.0.2) (68.1.2)
Installing collected packages: tzdata, sqlparse, asgiref, django
  Attempting uninstall: django
    Found existing installation: Django 4.2.10
    Uninstalling Django-4.2.10:
      Successfully uninstalled Django-4.2.10
Successfully installed asgiref-3.7.2 django-5.0.2 sqlparse-0.4.4 tzdata-2024.1

[notice] A new release of pip is available: 23.3.2 -> 24.0
[notice] To update, run: pip install --upgrade pip`

	got, err := filterPipInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
	if !strings.Contains(got, "installed 4 packages") {
		t.Errorf("expected 4 packages, got: %s", got)
	}
	if !strings.Contains(got, "asgiref-3.7.2") {
		t.Errorf("expected package names in output, got: %s", got)
	}
	// The "already satisfied" line should count toward upToDate but installed line takes precedence
	if strings.Contains(got, "up to date") {
		t.Errorf("should not show up-to-date when packages were installed, got: %s", got)
	}
}

func TestFilterPipInstall_ErrorsWithCollecting(t *testing.T) {
	// Errors need a pip install marker to pass the sanity guard
	raw := `Collecting nonexistent-package
ERROR: Could not find a version that satisfies the requirement nonexistent-package (from versions: none)
ERROR: No matching distribution found for nonexistent-package`

	got, err := filterPipInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ERROR: Could not find a version") {
		t.Errorf("expected first error preserved, got: %s", got)
	}
	if !strings.Contains(got, "ERROR: No matching distribution") {
		t.Errorf("expected second error preserved, got: %s", got)
	}
}

func TestFilterPipInstall_ErrorsOnlyNoMarker(t *testing.T) {
	// Without pip install markers, the filter passes through raw
	raw := `ERROR: Could not find a version that satisfies the requirement nonexistent-package (from versions: none)
ERROR: No matching distribution found for nonexistent-package`

	got, err := filterPipInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("expected raw passthrough without pip markers, got: %s", got)
	}
}

func TestFilterPipInstall_AlreadySatisfied(t *testing.T) {
	raw := `Requirement already satisfied: requests in /usr/lib/python3/dist-packages (2.31.0)
Requirement already satisfied: urllib3<3,>=1.21.1 in /usr/lib/python3/dist-packages (from requests) (2.2.0)
Requirement already satisfied: certifi>=2017.4.17 in /usr/lib/python3/dist-packages (from requests) (2024.2.2)
Requirement already satisfied: charset-normalizer<4,>=2 in /usr/lib/python3/dist-packages (from requests) (3.3.2)
Requirement already satisfied: idna<4,>=2.5 in /usr/lib/python3/dist-packages (from requests) (3.6)`

	got, err := filterPipInstall(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
	if !strings.Contains(got, "5 packages already up to date") {
		t.Errorf("expected 5 up to date, got: %s", got)
	}
}

func TestFilterPipList_Large(t *testing.T) {
	raw := `Package            Version
------------------ --------
asgiref            3.7.2
certifi            2024.2.2
charset-normalizer 3.3.2
Django             5.0.2
djangorestframework 3.14.0
gunicorn           21.2.0
idna               3.6
Pillow             10.2.0
psycopg2-binary    2.9.9
PyJWT              2.8.0
python-dotenv      1.0.1
pytz               2024.1
redis              5.0.1
requests           2.31.0
setuptools         68.1.2
sqlparse           0.4.4
tzdata             2024.1
urllib3             2.2.0
whitenoise         6.6.0`

	got, err := filterPipList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
	if !strings.Contains(got, "19 packages installed") {
		t.Errorf("expected 19 packages, got: %s", got)
	}
	// First 10 should be listed
	if !strings.Contains(got, "asgiref") {
		t.Errorf("expected first package listed, got: %s", got)
	}
	if !strings.Contains(got, "PyJWT") {
		t.Errorf("expected 10th package listed, got: %s", got)
	}
	// 11th+ should be in "and N more"
	if !strings.Contains(got, "and 9 more") {
		t.Errorf("expected 'and 9 more', got: %s", got)
	}
}

func TestFilterPipList_Empty(t *testing.T) {
	got, err := filterPipList("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
