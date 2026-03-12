package filters

import (
	"strings"
	"testing"
)

var npmViewFixture = `@angular/core@19.2.0 | MIT | deps: 3 | versions: 347
The core of Angular framework

dist.tarball:     https://registry.npmjs.org/@angular/core/-/@angular/core-19.2.0.tgz
dist.shasum:      abc123def456
dist.integrity:   sha512-longhashhere==
dist.unpackedSize: 1.2 MB

dependencies:
  rxjs:    ^6.5.3
  tslib:   ^2.3.0
  zone.js: ~0.15.0

maintainers:
- angular <devops+npm@angular.io>

dist-tags:
  latest: 19.2.0
  next:   19.3.0-rc.1

published 2 days ago by angular <devops+npm@angular.io>`

var npmViewSimpleFixture = `lodash@4.17.21 | MIT | deps: none | versions: 114
Lodash modular utilities.

dist.tarball:    https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz
dist.shasum:     679591c564c3bffaae8454cf0b3df370c3d6911c
dist.integrity:  sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZa2e6wi9wsr+nKbhcs==
dist.unpackedSize: 1.4 MB

dist-tags:
  latest: 4.17.21

published 3 years ago by jdalton <john.david.dalton@gmail.com>`

func TestNpmViewFull(t *testing.T) {
	got, err := filterNpmView(npmViewFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "@angular/core@19.2.0") {
		t.Errorf("expected package name in output: %s", got)
	}
	if !strings.Contains(got, "MIT") {
		t.Errorf("expected license in output: %s", got)
	}
	if !strings.Contains(got, "core of Angular") {
		t.Errorf("expected description in output: %s", got)
	}
	if !strings.Contains(got, "rxjs") {
		t.Errorf("expected dependency in output: %s", got)
	}
	if !strings.Contains(got, "19.2.0") {
		t.Errorf("expected latest dist-tag in output: %s", got)
	}

	// Noise should be stripped
	if strings.Contains(got, "tarball") {
		t.Errorf("tarball URL should be stripped: %s", got)
	}
	if strings.Contains(got, "shasum") {
		t.Errorf("shasum should be stripped: %s", got)
	}
	if strings.Contains(got, "integrity") {
		t.Errorf("integrity hash should be stripped: %s", got)
	}
	if strings.Contains(got, "maintainers") {
		t.Errorf("maintainers should be stripped: %s", got)
	}

	rawTokens := countTokens(npmViewFixture)
	filteredTokens := countTokens(got)
	savings := 100.0 - (float64(filteredTokens)/float64(rawTokens)*100.0)
	if savings < 40.0 {
		t.Errorf("expected >=40%% savings, got %.1f%%", savings)
	}
	t.Logf("token savings: %.1f%% (%d -> %d)", savings, rawTokens, filteredTokens)
	t.Logf("output:\n%s", got)
}

func TestNpmViewNoDeps(t *testing.T) {
	got, err := filterNpmView(npmViewSimpleFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "lodash@4.17.21") {
		t.Errorf("expected package name: %s", got)
	}
	if !strings.Contains(got, "Lodash modular") {
		t.Errorf("expected description: %s", got)
	}
	if !strings.Contains(got, "4.17.21") {
		t.Errorf("expected dist-tag: %s", got)
	}
	if strings.Contains(got, "tarball") || strings.Contains(got, "integrity") {
		t.Errorf("dist metadata should be stripped: %s", got)
	}

	t.Logf("output:\n%s", got)
}

func TestNpmViewNonView(t *testing.T) {
	raw := "some random output that is not npm view"
	got, err := filterNpmView(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("non-view output should pass through unchanged")
	}
}

func TestNpmUpdateRoutedToInstallFilter(t *testing.T) {
	raw := `
changed 3 packages, and audited 1234 packages in 8s

2 packages are looking for funding
  run ` + "`npm fund`" + ` for details

found 0 vulnerabilities
`
	f := getNpmFilter([]string{"update"})
	if f == nil {
		t.Fatal("expected filter for npm update, got nil")
	}
	got, err := f(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "changed 3 packages") {
		t.Errorf("expected changed count in output: %s", got)
	}
	if strings.Contains(got, "funding") {
		t.Errorf("funding notice should be stripped: %s", got)
	}
	t.Logf("output: %s", got)
}

func TestNpxPlaywrightNonTestSubcommandNotFiltered(t *testing.T) {
	f := getNpxFilter([]string{"playwright", "install"})
	if f != nil {
		t.Error("npx playwright install should not get a filter (install output is download progress, not test output)")
	}
}
