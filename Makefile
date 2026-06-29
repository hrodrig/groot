APP_NAME := groot
BIN_DIR := bin
DIST := dist
FREEBSD_ARCH ?= amd64
OPENBSD_ARCH ?= amd64
PORT_VERSION := $(shell cat VERSION 2>/dev/null | tr -d '\n\r' | sed 's/^v//')
IMAGE ?= $(APP_NAME):local
PLATFORMS ?= linux/amd64,linux/arm64
IMAGE_AMD64 ?= $(IMAGE)-amd64
IMAGE_ARM64 ?= $(IMAGE)-arm64
GOPATH_BIN := $(shell go env GOPATH)/bin
PREFIX ?= $(GOPATH_BIN)
BINDIR ?= $(PREFIX)
VERSION := $(shell cat VERSION 2>/dev/null | tr -d ' \n\r' || echo 0.1.0)
GIT_COMMIT := $(firstword $(shell git rev-parse --short HEAD 2>/dev/null | head -1))
ifeq ($(strip $(GIT_COMMIT)),)
  GIT_COMMIT := unknown
endif
GIT_BRANCH := $(firstword $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null | head -1))
ifeq ($(strip $(GIT_BRANCH)),)
  GIT_BRANCH := unknown
endif
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(GIT_COMMIT)' -X 'main.branch=$(GIT_BRANCH)' -X 'main.buildDate=$(BUILD_DATE)'

# Grype directory scan (see anchore/grype catalog rules for exclude globs).
GRYPE_FAIL_ON ?= high
GRYPE_DIR_EXCLUDES := --exclude './bin/**' --exclude './work/**' --exclude './dist/**'

# Release: optional strict image scan (see release-check).
STRICT_RELEASE ?= 0
TAG := v$(VERSION)

# Merged coverage gate for `cover` / `all` (default 80 percent merged statements; requires bc).
COVER_MIN ?= 80

# Passed to Dockerfile for -ldflags (same metadata as local go build).
DOCKER_BUILD_ARGS := \
	--build-arg APP_VERSION=$(VERSION) \
	--build-arg GIT_COMMIT=$(GIT_COMMIT) \
	--build-arg GIT_BRANCH=$(GIT_BRANCH) \
	--build-arg BUILD_DATE=$(BUILD_DATE)

.DEFAULT_GOAL := help

.PHONY: help all build test cover fmt fmt-check lint-fix lint vet run clean install install-kubectl-plugin docker-build docker-buildx docker-build-amd64 docker-build-arm64 scan govulncheck vulncheck ci gocyclo grype security release-check docker-scan test-e2e-kind e2e-kind snapshot dist-freebsd dist-openbsd port-freebsd-sync port-openbsd-sync

help:
	@echo "Available targets:"
	@echo "  make all            fmt, vet, test, gocyclo, cover, build"
	@echo "  make build          Build local binary"
	@echo "  make test-e2e-kind  E2E harness: Docker + kind + kubectl on host (groot uses client-go, not kubectl)"
	@echo "  make clean          Remove everything under bin/ except bin/.keep"
	@echo "  make cover          Merged coverage (coverage.out); gate with COVER_MIN (default 80; needs bc)"
	@echo "  make docker-build   Build container image with Docker"
	@echo "  make docker-buildx  Build multi-arch image (amd64, arm64) with Docker buildx"
	@echo "  make docker-scan    docker-build + Grype image scan (optional: STRICT_RELEASE=1 in release-check)"
	@echo "  make fmt            gofmt -w . (no simplify; use lint-fix for gofmt -s)"
	@echo "  make gocyclo        Fail if any function has cyclomatic complexity >= 15"
	@echo "  make govulncheck    govulncheck via go run (no install)"
	@echo "  make grype          Grype directory scan (excludes bin/work/dist; Docker fallback if grype missing)"
	@echo "  make help           Show this help"
	@echo "  make install        Install binary to GOPATH bin (user-writable)"
	@echo "  make install-kubectl-plugin  Install the kubectl-groot plugin binary (sets up kubectl plugin discovery)"
	@echo "  make lint           Run go vet"
	@echo "  make lint-fix       gofmt -s -w . (simplify with gofmt -s)"
	@echo "  make fmt-check      Fail if gofmt -s would change any file (same as CI)"
	@echo "  make release-check  VERSION semver + goreleaser + fmt-check + vet + cover + security"
	@echo "  make snapshot       Goreleaser snapshot to dist/ (no tag)"
	@echo "  make dist-freebsd   Tarball for FreeBSD ports (default FREEBSD_ARCH=amd64)"
	@echo "  make dist-openbsd   Tarball for OpenBSD ports (default OPENBSD_ARCH=amd64)"
	@echo "  make port-freebsd-sync   Set PORTVERSION in contrib/freebsd/Makefile from VERSION"
	@echo "  make port-openbsd-sync   Sync contrib/openbsd/port/Makefile from VERSION"
	@echo "  make run            Run collector with default config"
	@echo "  make scan           Build amd64/arm64 images and scan both with Grype"
	@echo "  make security       govulncheck + gocyclo + grype (dir scan)"
	@echo "  make test           Run Go tests (-count=1 -race, same as CI)"
	@echo "  make vet            go vet ./..."
	@echo "  make vulncheck      alias for govulncheck"
	@echo ""
	@echo "Variables:"
	@echo "  BINDIR=<dir>        Install bin dir (default: $(BINDIR))"
	@echo "  COVER_MIN=<n>       Minimum merged statement %% for cover/all (default: $(COVER_MIN); needs bc)"
	@echo "  GRYPE_FAIL_ON=      Grype severity gate (default: $(GRYPE_FAIL_ON))"
	@echo "  IMAGE=<name:tag>    Override image tag (default: $(IMAGE))"
	@echo "  PLATFORMS=<list>    buildx platforms (default: $(PLATFORMS))"
	@echo "  PREFIX=<dir>        Install prefix (default: $(PREFIX))"
	@echo "  STRICT_RELEASE=1    release-check also runs docker-scan (default: $(STRICT_RELEASE))"

all: fmt vet test gocyclo cover build

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/groot

test:
	go test -count=1 ./... -race

# Merged report across all packages. When COVER_MIN > 0, fails if total statement coverage is below that value (needs bc).
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

# Same check as .github/workflows/ci.yml (lint job, formatting step).
fmt-check:
	@out=$$(gofmt -s -l .); \
	if [ -n "$$out" ]; then \
		echo "Run: make lint-fix"; \
		echo "$$out"; \
		exit 1; \
	fi

lint:
	go vet ./...

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

# kubectl plugin install: same `make build` but lays the binary down as
# `kubectl-groot` and on a directory that kubectl's plugin discovery will
# actually find (`$(PREFIX)` when in PATH, else warn explicitly). Detect
# a `make install PREFIX=...` that puts the binary somewhere unusual and
# refuse rather than silently dropping a plugin nobody can run.
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
	docker build $(DOCKER_BUILD_ARGS) -t $(IMAGE) .

docker-buildx:
	docker buildx build $(DOCKER_BUILD_ARGS) --platform $(PLATFORMS) -t $(IMAGE) .

docker-build-amd64:
	docker buildx build $(DOCKER_BUILD_ARGS) --platform linux/amd64 --load -t $(IMAGE_AMD64) .

docker-build-arm64:
	docker buildx build $(DOCKER_BUILD_ARGS) --platform linux/arm64 --load -t $(IMAGE_ARM64) .

scan: docker-build-amd64 docker-build-arm64
	@command -v grype >/dev/null 2>&1 || (echo "grype is required. Install: https://github.com/anchore/grype" && exit 1)
	grype docker:$(IMAGE_AMD64)
	grype docker:$(IMAGE_ARM64)

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

vulncheck: govulncheck

ci: fmt-check lint gocyclo test
	@echo "OK: ci (fmt-check, vet, gocyclo, test)"

# Disposable kind cluster + log workload + groot collect (same layout as pgwd: testing/scripts + testing/k8s).
test-e2e-kind: build
	@command -v kind >/dev/null 2>&1 || { echo "kind is required: https://kind.sigs.k8s.io/"; exit 1; }
	@command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required for test-e2e-kind (apply/wait); groot itself does not need kubectl"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "docker is required"; exit 1; }
	@chmod +x testing/scripts/test-e2e-kind.sh
	@./testing/scripts/test-e2e-kind.sh

e2e-kind: test-e2e-kind

# Fail if any function has cyclomatic complexity >= 15 (gocyclo: complexity > 14).
gocyclo:
	go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -over 14 .

# Directory scan with path exclusions; Docker fallback via anchore/grype image if grype is not installed.
grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v "$(PWD):/workspace" anchore/grype:latest \
			dir:/workspace $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	fi

security: govulncheck gocyclo grype
	@echo "OK: security (govulncheck, gocyclo, grype)"

# Semver + goreleaser + fmt-check (CI parity) + vet + cover + security (+ optional docker-scan when STRICT_RELEASE=1).
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

# Image scan after docker-build; --pull=always avoids a stale local anchore/grype:latest cache.
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

port-freebsd-sync:
	@[ -n "$(PORT_VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@sed -i.bak "s/^PORTVERSION=.*/PORTVERSION=\t$(PORT_VERSION)/" contrib/freebsd/Makefile
	@rm -f contrib/freebsd/Makefile.bak
	@echo "Updated contrib/freebsd/Makefile PORTVERSION to $(PORT_VERSION)"

port-openbsd-sync:
	@[ -n "$(PORT_VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@test -f contrib/openbsd/port/Makefile || { echo "Error: contrib/openbsd/port/Makefile not found"; exit 1; }
	@sed -i.bak \
	  -e 's#^DISTNAME =.*#DISTNAME =	groot_v$(PORT_VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}#' \
	  -e 's#^PKGNAME =.*#PKGNAME =	groot-$(PORT_VERSION)#' \
	  -e 's#^MASTER_SITES =.*#MASTER_SITES =	https://github.com/hrodrig/groot/releases/download/v$(PORT_VERSION)/#' \
	  -e 's#^DISTFILES =.*#DISTFILES =	groot_v$(PORT_VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz#' \
	  contrib/openbsd/port/Makefile
	@rm -f contrib/openbsd/port/Makefile.bak
	@echo "Updated contrib/openbsd/port/Makefile to $(PORT_VERSION)"

dist-freebsd:
	@set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	echo "$(FREEBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: FREEBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(FREEBSD_ARCH)"; \
	out="$(DIST)/groot_v$${ver}_freebsd_$$arch.tar.gz"; \
	stage="/tmp/groot-dist-root-$$PPID"; \
	tmpbin="$(DIST)/groot-freebsd-$$arch-$$PPID"; \
	echo "Building groot for FreeBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=freebsd GOARCH="$$arch" go build -trimpath -ldflags "$(LDFLAGS)" -o "$$tmpbin" ./cmd/groot; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/doc/groot" "$$stage/share/examples/groot"; \
	cp "$$tmpbin" "$$stage/groot"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/groot/LICENSE"; \
	cp configs/groot.yml.sample "$$stage/share/examples/groot/groot.yml.sample"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"

dist-openbsd:
	@set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	echo "$(OPENBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: OPENBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(OPENBSD_ARCH)"; \
	out="$(DIST)/groot_v$${ver}_openbsd_$$arch.tar.gz"; \
	stage="/tmp/groot-openbsd-dist-root-$$PPID"; \
	tmpbin="$(DIST)/groot-openbsd-$$arch-$$PPID"; \
	echo "Building groot for OpenBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=openbsd GOARCH="$$arch" go build -trimpath -ldflags "$(LDFLAGS)" -o "$$tmpbin" ./cmd/groot; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/doc/groot" "$$stage/share/examples/groot"; \
	cp "$$tmpbin" "$$stage/groot"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/groot/LICENSE"; \
	cp configs/groot.yml.sample "$$stage/share/examples/groot/groot.yml.sample"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"
