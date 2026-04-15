# DO NOT EDIT. Generated with:
#
#    devctl
#
#    https://github.com/giantswarm/devctl/blob/6a704f7e2a8b0f09e82b5bab88f17971af849711/pkg/gen/input/makefile/internal/file/Makefile.template
#

# include Makefile.*.mk (commented out as these files don't exist)

# CAPI version to use for downloading CRDs
export CAPI_VERSION ?= v1.11.2

# CAPA (AWS provider) version to use for downloading CRDs
export CAPA_VERSION ?= v2.9.2

# CAPZ (Azure provider) version to use for downloading CRDs
export CAPZ_VERSION ?= v1.21.1

# CAPV (vSphere provider) version to use for downloading CRDs
export CAPV_VERSION ?= v1.14.0

# CAPVCD (Cloud Director provider) version to use for downloading CRDs
export CAPVCD_VERSION ?= v1.3.2

# CAPG (GCP provider) version to use for downloading CRDs
export CAPG_VERSION ?= v1.10.0

# Maximum parallel test goroutines within each test binary.
# Each parallel subtest acquires its own k8senv instance (kine + apiserver),
# so this controls the peak number of concurrent API servers.
TEST_PARALLEL ?= 4

# Temporary directory for test artifacts (Go build cache, k8senv data, etc.).
# Defaults to .tmp/ in the project root to avoid filling a small /tmp tmpfs.
export TEST_TMPDIR := $(CURDIR)/.tmp

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z%\\\/_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Setup

.PHONY: download-crds
download-crds: ## Download all CRDs from upstream
	@./scripts/download-crds.sh --all

.PHONY: download-capi-crds
download-capi-crds: ## Download CAPI CRDs (use CAPI_VERSION to override)
	@./scripts/download-crds.sh capi

.PHONY: download-capa-crds
download-capa-crds: ## Download CAPA CRDs (use CAPA_VERSION to override)
	@./scripts/download-crds.sh capa

.PHONY: download-capz-crds
download-capz-crds: ## Download CAPZ CRDs (use CAPZ_VERSION to override)
	@./scripts/download-crds.sh capz

.PHONY: download-capv-crds
download-capv-crds: ## Download CAPV CRDs (use CAPV_VERSION to override)
	@./scripts/download-crds.sh capv

.PHONY: download-capvcd-crds
download-capvcd-crds: ## Download CAPVCD CRDs (use CAPVCD_VERSION to override)
	@./scripts/download-crds.sh capvcd

.PHONY: download-capg-crds
download-capg-crds: ## Download CAPG CRDs (use CAPG_VERSION to override)
	@./scripts/download-crds.sh capg

##@ Build

.PHONY: build
build: ## Build the binary for the current platform
	go build -o mcp-capi

.PHONY: install
install: build ## Install the binary
	mv mcp-capi $(GOPATH)/bin/mcp-capi

##@ Release

.PHONY: release-dry-run
release-dry-run: ## Test the release process without publishing
	goreleaser release --snapshot --clean --skip=announce,publish,validate

.PHONY: release-local
release-local: ## Create a release locally
	goreleaser release --clean

##@ Development

.PHONY: lint-yaml
lint-yaml: ## Run YAML linter
	@echo "Running YAML linter..."
	@# Exclude zz_generated files
	@yamllint .github/workflows/auto-release.yaml .github/workflows/ci.yaml .goreleaser.yaml

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

.PHONY: check
check: lint-yaml lint test ## Run all linters (YAML + Go)

##@ Testing

.PHONY: test
test: ## Run all tests (unit + integration)
	@echo "Running all tests..."
	@mkdir -p $(TEST_TMPDIR)
	@TMPDIR=$(TEST_TMPDIR) \
	go test -parallel=$(TEST_PARALLEL) -count=1 -v ./...

.PHONY: test-coverage
test-coverage: ## Run all tests with coverage
	@echo "Running all tests with coverage..."
	@mkdir -p $(TEST_TMPDIR)
	@TMPDIR=$(TEST_TMPDIR) \
	go test -parallel=$(TEST_PARALLEL) -count=1 -v \
		-coverprofile=coverage.out \
		-covermode=atomic \
		-coverpkg=./... \
		./...

.PHONY: test-single
test-single: ## Run a single test (use FOCUS="pattern")
	@echo "Running focused test..."
	@mkdir -p $(TEST_TMPDIR)
	@TMPDIR=$(TEST_TMPDIR) \
	go test -parallel=$(TEST_PARALLEL) -count=1 -v \
		-run "$(FOCUS)" \
		./...

# Note: These targets require Docker and 'act' to be installed.
# See: https://github.com/nektos/act#installation

.PHONY: test-ci-pr
test-ci-pr: ## Run 'act' to simulate CI checks for a pull request
	@echo "Simulating CI workflow (pull_request event)..."
	@act pull_request --job check

.PHONY: test-ci-push
test-ci-push: ## Run 'act' to simulate CI checks for a push to main
	@echo "Simulating CI workflow (push event)..."
	@act push --job check

.PHONY: test-auto-release
test-auto-release: ## Run 'act' to simulate the auto-release workflow
	@echo "Simulating Auto-Release workflow (merged pull_request event)..."
	@echo "NOTE: Requires 'merged_pr_event.json' in the project root."
	@echo "NOTE: Git push steps within the workflow are expected to fail locally."
	@act pull_request --job auto_release --eventpath merged_pr_event.json
