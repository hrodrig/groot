APP_NAME := groot
BIN_DIR := bin
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

.PHONY: help all build test cover fmt lint-fix lint vet run clean install docker-build docker-buildx docker-build-amd64 docker-build-arm64 scan govulncheck vulncheck ci gocyclo grype security release-check docker-scan

help:
	@echo "Available targets:"
	@echo "  make all            fmt, vet, test, gocyclo, cover, build"
	@echo "  make build          Build local binary"
	@echo "  make ci             Run lint and tests (same bundle as local CI)"
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
	@echo "  make lint           Run go vet"
	@echo "  make lint-fix       gofmt -s -w . (simplify with gofmt -s)"
	@echo "  make release-check  VERSION semver + goreleaser check + lint + cover + security"
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
			echo "  export PATH=\"$(BINDIR):\$$PATH\""; \
			;; \
	esac

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

ci: lint test
	@echo "OK: lint, test"

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

# Semver + goreleaser check + lint + cover + security (+ optional docker-scan when STRICT_RELEASE=1).
release-check:
	@test -f VERSION || { echo "VERSION file is required"; exit 1; }
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1; }
	@git rev-parse --git-dir >/dev/null 2>&1 || { echo "release-check requires a git repository (clone or: git init && git remote add origin <url>)"; exit 1; }
	@git remote get-url origin >/dev/null 2>&1 || { echo "release-check requires git remote origin (GoReleaser scm validation)"; exit 1; }
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1; }
	goreleaser check
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
