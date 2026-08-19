# groot — build, test, release (GNU Make). On FreeBSD use gmake (pkg install gmake).
# GNU Make prefers GNUmakefile over Makefile; the Makefile stub forwards to gmake.

APP_NAME := groot
BIN_DIR := bin
DIST    := dist
MODULE  := github.com/hrodrig/groot

# Single source of truth: VERSION file at repo root.
VERSION_RAW ?= $(shell cat VERSION 2>/dev/null | tr -d '\n\r')
VERSION     := $(patsubst v%,%,$(VERSION_RAW))
TAG         := v$(VERSION)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BRANCH      := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
BUILDDATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.branch=$(BRANCH)' \
	-X 'main.buildDate=$(BUILDDATE)'

# Fails early when Docker is required but not running.
check-docker = @docker info >/dev/null 2>&1 || { echo "Error: Docker is not running. Start Docker and try again."; exit 1; }

# Grype directory scan exclusions (see anchore/grype catalog rules for globs).
GRYPE_FAIL_ON ?= high
GRYPE_DIR_EXCLUDES := --exclude './bin/**' --exclude './work/**' --exclude './dist/**' --exclude './.tmp-go/**'

# Pin golangci-lint for reproducible `make lint` / CI (config is v2).
# Must be ≥ v2.9.0 so the published binary is built with Go ≥ 1.26 (matches go.mod).
# Family pin: groot-trigger / groot-share use v2.12.2.
GOLANGCI_LINT_VERSION ?= v2.12.2

# Release: optional strict image scan (see release-check).
STRICT_RELEASE ?= 0

# Merged coverage gate (default 80 percent; needs bc).
COVER_MIN ?= 80

# BSD dist tarball arch (cross-compile).
FREEBSD_ARCH ?= amd64
OPENBSD_ARCH ?= amd64

# Docker image tag and platforms.
IMAGE       ?= $(APP_NAME):local
PLATFORMS   ?= linux/amd64,linux/arm64
IMAGE_AMD64 ?= $(IMAGE)-amd64
IMAGE_ARM64 ?= $(IMAGE)-arm64

# Install paths.
PREFIX  ?= $(shell go env GOPATH)/bin
BINDIR  ?= $(PREFIX)

# Docker build args (same metadata as local go build).
DOCKER_BUILD_ARGS := \
	--build-arg APP_VERSION=$(VERSION) \
	--build-arg GIT_COMMIT=$(COMMIT) \
	--build-arg GIT_BRANCH=$(BRANCH) \
	--build-arg BUILD_DATE=$(BUILDDATE)

# --- Colors (same pattern as pgwd, kzero, gghstats) ---
GREEN  := \033[0;32m
YELLOW := \033[0;33m
CYAN   := \033[0;36m
RESET  := \033[0m
ifneq ($(NO_COLOR),)
  GREEN  :=
  YELLOW :=
  CYAN   :=
  RESET  :=
endif

.DEFAULT_GOAL := help

.PHONY: help all build test cover fmt fmt-check lint-fix lint vet run clean install install-kubectl-plugin docker-build docker-buildx docker-build-amd64 docker-build-arm64 scan govulncheck vulncheck ci gocyclo grype security release-check docker-scan test-e2e-kind e2e-kind snapshot dist-freebsd dist-openbsd port-freebsd-sync port-openbsd-sync man-sync

help:
	@echo "$(GREEN)groot$(RESET) — Kubernetes observability collector"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(YELLOW)Build:$(RESET)"
	@echo "  $(GREEN)build$(RESET)                     Build local binary ($(BIN_DIR)/$(APP_NAME))"
	@echo ""
	@echo "$(YELLOW)Install & clean:$(RESET)"
	@echo "  $(GREEN)install$(RESET)                  Install binary to GOPATH bin (user-writable)"
	@echo "  $(GREEN)install-kubectl-plugin$(RESET)   Install kubectl-groot plugin binary"
	@echo "  $(GREEN)clean$(RESET)                    Remove everything under $(BIN_DIR)/ except $(BIN_DIR)/.keep"
	@echo ""
	@echo "$(YELLOW)Run:$(RESET)"
	@echo "  $(GREEN)run$(RESET)                      Run collector with default config (configs/groot.yml.sample)"
	@echo ""
	@echo "$(YELLOW)Test:$(RESET)"
	@echo "  $(GREEN)test$(RESET)                     Run Go tests (-count=1 -race, same as CI)"
	@echo "  $(GREEN)test-e2e-kind$(RESET)            E2E harness: Docker + kind + kubectl on host"
	@echo "  $(GREEN)cover$(RESET)                    Merged coverage (coverage.out); gate with COVER_MIN (default $(COVER_MIN); needs bc)"
	@echo ""
	@echo "$(YELLOW)Quality:$(RESET)"
	@echo "  $(GREEN)fmt$(RESET)                      gofmt -w . (no simplify; use lint-fix for gofmt -s)"
	@echo "  $(GREEN)fmt-check$(RESET)                Fail if gofmt -s would change any file (same as CI)"
	@echo "  $(GREEN)lint-fix$(RESET)                 gofmt -s -w ."
	@echo "  $(GREEN)lint$(RESET)                     golangci-lint run (govet/staticcheck/…; pin GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION))"
	@echo "  $(GREEN)vet$(RESET)                      go vet ./..."
	@echo "  $(GREEN)gocyclo$(RESET)                  Fail if any function has cyclomatic complexity >= 15"
	@echo "  $(GREEN)govulncheck$(RESET)              govulncheck via go run (no install)"
	@echo "  $(GREEN)grype$(RESET)                    Grype directory scan (excludes bin/work/dist; Docker fallback)"
	@echo "  $(GREEN)security$(RESET)                 govulncheck + gocyclo + grype (dir scan)"
	@echo ""
	@echo "$(YELLOW)Docker:$(RESET)"
	@echo "  $(GREEN)docker-build$(RESET)             Build container image ($(IMAGE))"
	@echo "  $(GREEN)docker-buildx$(RESET)            Build multi-arch image (amd64, arm64)"
	@echo "  $(GREEN)docker-scan$(RESET)              docker-build + Grype image scan"
	@echo "  $(GREEN)scan$(RESET)                     Build amd64/arm64 images and scan both with Grype"
	@echo ""
	@echo "$(YELLOW)Release:$(RESET)"
	@echo "  $(GREEN)release-check$(RESET)            VERSION semver + goreleaser + fmt-check + golangci-lint + cover + security"
	@echo "  $(GREEN)snapshot$(RESET)                 Goreleaser snapshot to $(DIST)/ (no tag)"
	@echo "  $(GREEN)dist-freebsd$(RESET)             Tarball for FreeBSD ports (default FREEBSD_ARCH=$(FREEBSD_ARCH))"
	@echo "  $(GREEN)dist-openbsd$(RESET)             Tarball for OpenBSD ports (default OPENBSD_ARCH=$(OPENBSD_ARCH))"
	@echo "  $(GREEN)port-freebsd-sync$(RESET)        Set PORTVERSION in contrib/freebsd/Makefile from VERSION"
	@echo "  $(GREEN)port-openbsd-sync$(RESET)        Sync contrib/openbsd/port/Makefile from VERSION"
	@echo "  $(GREEN)man-sync$(RESET)                 Bump .TH in contrib/man/man1/*.1 from VERSION + today"
	@echo ""
	@echo "$(CYAN)Current version:$(RESET) $$(cat VERSION 2>/dev/null | tr -d '\n\r' || echo '?') (ldflags $(VERSION), branch $(BRANCH))"
	@echo ""
	@echo "$(CYAN)Variables:$(RESET)"
	@echo "  BINDIR=<dir>        Install bin dir (default: $(BINDIR))"
	@echo "  COVER_MIN=<n>       Minimum merged statement % for cover/all (default: $(COVER_MIN); needs bc)"
	@echo "  GRYPE_FAIL_ON=      Grype severity gate (default: $(GRYPE_FAIL_ON))"
	@echo "  IMAGE=<name:tag>    Override image tag (default: $(IMAGE))"
	@echo "  PLATFORMS=<list>    buildx platforms (default: $(PLATFORMS))"
	@echo "  PREFIX=<dir>        Install prefix (default: $(PREFIX))"
	@echo "  STRICT_RELEASE=1    release-check also runs docker-scan (default: $(STRICT_RELEASE))"
	@echo "  NO_COLOR=1          Disable colored output"
	@echo ""
	@echo "$(CYAN)Examples:$(RESET)"
	@echo "  make build"
	@echo "  make test"
	@echo "  make release-check"
	@echo "  NO_COLOR=1 make help"

all: fmt vet test gocyclo cover build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/groot

test:
	go test -count=1 ./... -race

# Merged report across all packages. Fails if total statement coverage is below COVER_MIN (needs bc).
cover:
	go test -count=1 ./... -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
	@P=$$(go tool cover -func=coverage.out | tail -1 | sed 's/^.*[[:space:]]\([0-9.]*\)%.*/\1/'); \
		echo "total (merged) statement coverage: $$P% (minimum $(COVER_MIN)%)"; \
		if [ "$(COVER_MIN)" -gt 0 ]; then \
			command -v bc >/dev/null 2>&1 || { echo "COVER_MIN>0 requires bc (e.g. apt install bc)"; exit 1; }; \
			if [ "$$(echo "$$P < $(COVER_MIN)" | bc)" -eq 1 ]; then \
				echo "coverage below $(COVER_MIN)% — add tests or lower COVER_MIN"; exit 1; \
			fi; \
		fi

fmt:
	gofmt -w .

lint-fix:
	gofmt -s -w .

# Same check as CI (lint job, formatting step).
fmt-check:
	@out=$$(gofmt -s -l .); \
	if [ -n "$$out" ]; then \
		echo "Run: make lint-fix"; \
		echo "$$out"; \
		exit 1; \
	fi

# golangci-lint v2 (.golangci.yml). Prefer binary on PATH; else go run pinned module.
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...; \
	fi

vet:
	go vet ./...

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/groot collect --config configs/groot.yml.sample

clean:
	@mkdir -p "$(BIN_DIR)"
	@for f in $$(find "$(BIN_DIR)" -mindepth 1 -maxdepth 1 ! -name '.keep' 2>/dev/null); do rm -rf "$$f"; done

install: build
	install -d "$(BINDIR)"
	install -m 755 $(BIN_DIR)/$(APP_NAME) "$(BINDIR)/$(APP_NAME)"
	@case ":$$PATH:" in \
		*":$(BINDIR):"*) \
			echo "Installed $(APP_NAME) to $(BINDIR) (already in PATH)."; \
			;; \
		*) \
			echo "Installed $(APP_NAME) to $(BINDIR)."; \
			echo "Warning: $(BINDIR) is not in PATH."; \
			echo "Add this to your shell profile:"; \
			echo "  export PATH=\"$(BINDIR):$$PATH\""; \
			;; \
	esac

# kubectl plugin install: lays binary down as kubectl-groot in a PATH directory.
install-kubectl-plugin: build
	@case ":$$PATH:" in \
		*":$(BINDIR):"*) \
			;; \
		*) \
			echo "Error: $(BINDIR) is not in PATH."; \
			echo "kubectl plugin discovery walks every directory in PATH and looks for executables starting with 'kubectl-'."; \
			echo "Set PREFIX to a directory that IS in PATH (default: $(PREFIX))."; \
			exit 1; \
			;; \
	esac
	install -m 755 $(BIN_DIR)/$(APP_NAME) "$(BINDIR)/kubectl-groot"
	@echo "Installed kubectl-groot to $(BINDIR)."
	@echo "Verify with: kubectl plugin list | grep groot"

docker-build:
	$(check-docker)
	docker build $(DOCKER_BUILD_ARGS) -t $(IMAGE) .

docker-buildx:
	$(check-docker)
	docker buildx build $(DOCKER_BUILD_ARGS) --platform $(PLATFORMS) -t $(IMAGE) .

docker-build-amd64:
	$(check-docker)
	docker buildx build $(DOCKER_BUILD_ARGS) --platform linux/amd64 --load -t $(IMAGE_AMD64) .

docker-build-arm64:
	$(check-docker)
	docker buildx build $(DOCKER_BUILD_ARGS) --platform linux/arm64 --load -t $(IMAGE_ARM64) .

scan: docker-build-amd64 docker-build-arm64
	@command -v grype >/dev/null 2>&1 || (echo "grype is required. Install: https://github.com/anchore/grype" && exit 1)
	grype docker:$(IMAGE_AMD64)
	grype docker:$(IMAGE_ARM64)

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

vulncheck: govulncheck

ci: fmt-check lint gocyclo test
	@echo "OK: ci (fmt-check, golangci-lint, gocyclo, test)"

gocyclo:
	go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -over 14 .

# Directory scan with path exclusions; Docker fallback via anchore/grype image.
grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v "$(CURDIR):/workspace" anchore/grype:latest \
			dir:/workspace $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	fi

security: govulncheck gocyclo grype
	@echo "OK: security (govulncheck, gocyclo, grype)"

# E2E test with kind cluster.
test-e2e-kind: build
	@command -v kind >/dev/null 2>&1 || { echo "kind is required: https://kind.sigs.k8s.io/"; exit 1; }
	@command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required for test-e2e-kind (apply/wait); groot itself does not need kubectl"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "docker is required"; exit 1; }
	@chmod +x testing/scripts/test-e2e-kind.sh
	@./testing/scripts/test-e2e-kind.sh

e2e-kind: test-e2e-kind

# Semver + goreleaser + fmt-check (CI parity) + golangci-lint + cover + security (+ optional docker-scan when STRICT_RELEASE=1).
release-check:
	@test -f VERSION || { echo "VERSION file is required"; exit 1; }
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1; }
	@git rev-parse --git-dir >/dev/null 2>&1 || { echo "release-check requires a git repository (clone or: git init && git remote add origin <url>)"; exit 1; }
	@git remote get-url origin >/dev/null 2>&1 || { echo "release-check requires git remote origin (GoReleaser scm validation)"; exit 1; }
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1; }
	goreleaser check
	@$(MAKE) fmt-check
	@$(MAKE) lint
	@$(MAKE) cover
	@$(MAKE) security
	@if [ "$(STRICT_RELEASE)" = "1" ]; then \
		echo "STRICT_RELEASE=1 -> running docker-scan"; \
		$(MAKE) docker-scan; \
	else \
		echo "STRICT_RELEASE=0 -> skipping docker-scan"; \
	fi
	@echo "All release checks passed."

# Image scan after docker-build; --pull=always avoids stale local anchore/grype:latest cache.
docker-scan: docker-build
	@if command -v grype >/dev/null 2>&1; then \
		grype $(IMAGE) --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest \
			$(IMAGE) --fail-on $(GRYPE_FAIL_ON); \
	fi

snapshot:
	@ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required for snapshot"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	goreleaser release --snapshot --clean

man-sync:
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@today=$$(date +%Y-%m-%d); \
	for f in contrib/man/man1/groot.1 contrib/man/man1/kubectl-groot.1; do \
	  test -f "$$f" || { echo "Error: $$f not found"; exit 1; }; \
	  name=$$(awk '/^\.TH /{print $$2; exit}' "$$f"); \
	  [ -n "$$name" ] || { echo "Error: no .TH line in $$f"; exit 1; }; \
	  sed -i.bak "s/^\.TH .*/.TH $$name 1 \"$$today\" \"groot v$(VERSION)\" \"User Commands\"/" "$$f"; \
	  rm -f "$$f.bak"; \
	  echo "Updated $$f .TH to groot v$(VERSION) ($$today)"; \
	done

port-freebsd-sync:
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@sed -i.bak "s/^PORTVERSION=.*/PORTVERSION=\t$(VERSION)/" contrib/freebsd/Makefile
	@rm -f contrib/freebsd/Makefile.bak
	@echo "Updated contrib/freebsd/Makefile PORTVERSION to $(VERSION)"

port-openbsd-sync:
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@test -f contrib/openbsd/port/Makefile || { echo "Error: contrib/openbsd/port/Makefile not found"; exit 1; }
	@sed -i.bak \
	  -e 's#^DISTNAME =.*#DISTNAME =\tgroot_v$(VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}#' \
	  -e 's#^PKGNAME =.*#PKGNAME =\tgroot-$(VERSION)#' \
	  -e 's#^MASTER_SITES =.*#MASTER_SITES =\thttps://github.com/hrodrig/groot/releases/download/v$(VERSION)/#' \
	  -e 's#^DISTFILES =.*#DISTFILES =\tgroot_v$(VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz#' \
	  contrib/openbsd/port/Makefile
	@rm -f contrib/openbsd/port/Makefile.bak
	@echo "Updated contrib/openbsd/port/Makefile to $(VERSION)"

dist-freebsd:
	@set -e; \
	ver="$(VERSION)"; \
	[ -n "$$ver" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver)"; exit 1; }; \
	echo "$(FREEBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: FREEBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(FREEBSD_ARCH)"; \
	out="$(DIST)/groot_v$${ver}_freebsd_$$arch.tar.gz"; \
	stage="/tmp/groot-dist-root-$$PPID"; \
	tmpbin="$(DIST)/groot-freebsd-$$arch-$$PPID"; \
	echo "Building groot for FreeBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=freebsd GOARCH="$$arch" go build -trimpath -ldflags "$(LDFLAGS)" -o "$$tmpbin" ./cmd/groot; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/doc/groot" "$$stage/share/examples/groot" "$$stage/share/man/man1"; \
	cp "$$tmpbin" "$$stage/groot"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/groot/LICENSE"; \
	cp configs/groot.yml.sample "$$stage/share/examples/groot/groot.yml.sample"; \
	cp contrib/man/man1/groot.1 "$$stage/share/man/man1/groot.1"; \
	cp contrib/man/man1/kubectl-groot.1 "$$stage/share/man/man1/kubectl-groot.1"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"

dist-openbsd:
	@set -e; \
	ver="$(VERSION)"; \
	[ -n "$$ver" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver)"; exit 1; }; \
	echo "$(OPENBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: OPENBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(OPENBSD_ARCH)"; \
	out="$(DIST)/groot_v$${ver}_openbsd_$$arch.tar.gz"; \
	stage="/tmp/groot-openbsd-dist-root-$$PPID"; \
	tmpbin="$(DIST)/groot-openbsd-$$arch-$$PPID"; \
	echo "Building groot for OpenBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=openbsd GOARCH="$$arch" go build -trimpath -ldflags "$(LDFLAGS)" -o "$$tmpbin" ./cmd/groot; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/doc/groot" "$$stage/share/examples/groot" "$$stage/share/man/man1"; \
	cp "$$tmpbin" "$$stage/groot"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/groot/LICENSE"; \
	cp configs/groot.yml.sample "$$stage/share/examples/groot/groot.yml.sample"; \
	cp contrib/man/man1/groot.1 "$$stage/share/man/man1/groot.1"; \
	cp contrib/man/man1/kubectl-groot.1 "$$stage/share/man/man1/kubectl-groot.1"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"
