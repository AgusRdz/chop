.PHONY: build test clean cross install changelog release release-patch release-minor release-major

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	docker compose run --rm dev go build -ldflags="$(LDFLAGS)" -o bin/chop .

test:
	docker compose run --rm dev go test ./... -v

clean:
	rm -rf bin/

UNAME_S := $(shell uname -s)
ifeq ($(findstring MINGW,$(UNAME_S)),MINGW)
  GOOS ?= windows
else ifeq ($(findstring MSYS,$(UNAME_S)),MSYS)
  GOOS ?= windows
else ifeq ($(findstring Darwin,$(UNAME_S)),Darwin)
  GOOS ?= darwin
else
  GOOS ?= linux
endif
GOARCH ?= $(if $(filter arm64 aarch64,$(shell uname -m)),arm64,amd64)
EXT := $(if $(filter windows,$(GOOS)),.exe,)
BINARY := bin/chop$(EXT)

ifeq ($(GOOS),windows)
  INSTALL_DIR ?= $(LOCALAPPDATA)/Programs/chop
else
  INSTALL_DIR ?= $(HOME)/.local/bin
endif

install:
	docker compose run --rm dev sh -c "CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags='$(LDFLAGS)' -o $(BINARY) ."
	@mkdir -p "$(INSTALL_DIR)"
	cp $(BINARY) "$(INSTALL_DIR)/chop$(EXT)"
	@echo "installed chop $(VERSION) ($(GOOS)/$(GOARCH)) to $(INSTALL_DIR)/chop$(EXT)"

# --- Changelog ---
# Requires: git-cliff (https://git-cliff.org)
changelog:
	git-cliff --output CHANGELOG.md
	@echo "updated CHANGELOG.md"

# --- Release helpers ---
# Usage: make release          (auto-detect bump from commits)
#        make release-patch    (v1.6.0 -> v1.6.1)
#        make release-minor    (v1.6.0 -> v1.7.0)
#        make release-major    (v1.6.0 -> v2.0.0)
CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)
MAJOR := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f1)
MINOR := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f3)

# Auto-detect release type from conventional commits since last tag
release:
	@BUMP=patch; \
	if git log $$(git describe --tags --abbrev=0)..HEAD --format="%s" | grep -qE '^feat(\(.*\))?!:'; then BUMP=major; \
	elif git log $$(git describe --tags --abbrev=0)..HEAD --format="%B" | grep -q 'BREAKING CHANGE'; then BUMP=major; \
	elif git log $$(git describe --tags --abbrev=0)..HEAD --format="%s" | grep -qE '^feat'; then BUMP=minor; fi; \
	echo "detected: $$BUMP"; \
	$(MAKE) release-$$BUMP

release-patch:
	@NEXT=v$(MAJOR).$(MINOR).$(shell echo $$(($(PATCH)+1))); \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git-cliff --tag $$NEXT --output CHANGELOG.md && \
	git add CHANGELOG.md && \
	git commit -m "chore: update changelog for $$NEXT" && \
	git tag $$NEXT && git push origin HEAD $$NEXT && echo "released $$NEXT"

release-minor:
	@NEXT=v$(MAJOR).$(shell echo $$(($(MINOR)+1))).0; \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git-cliff --tag $$NEXT --output CHANGELOG.md && \
	git add CHANGELOG.md && \
	git commit -m "chore: update changelog for $$NEXT" && \
	git tag $$NEXT && git push origin HEAD $$NEXT && echo "released $$NEXT"

release-major:
	@NEXT=v$(shell echo $$(($(MAJOR)+1))).0.0; \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git-cliff --tag $$NEXT --output CHANGELOG.md && \
	git add CHANGELOG.md && \
	git commit -m "chore: update changelog for $$NEXT" && \
	git tag $$NEXT && git push origin HEAD $$NEXT && echo "released $$NEXT"

cross:
	docker compose run --rm dev sh -c "\
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/chop-linux-amd64 . && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/chop-darwin-amd64 . && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='$(LDFLAGS)' -o bin/chop-darwin-arm64 . && \
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/chop-windows-amd64.exe ."
