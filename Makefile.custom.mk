KINE_VERSION := v0.14.12
KUBE_APISERVER_VERSION := v1.35.2

# The architect go-build job runs `make test` (test_target: test). The kine +
# kube-apiserver integration suite skips (os.Exit 0) unless those binaries are
# on PATH, so install them as a `make test` prerequisite -- this used to be a
# separate ci.yaml step. Only adds a prerequisite; the generated `go test`
# recipe in Makefile.gen.go.mk is not overridden.
test: install-test-binaries

.PHONY: install-test-binaries
install-test-binaries: ## Install kine and kube-apiserver needed by integration tests (Linux amd64).
	mkdir -p "$(shell go env GOPATH)/bin"
	curl -fsSL "https://github.com/k3s-io/kine/releases/download/$(KINE_VERSION)/kine-amd64" \
		-o "$(shell go env GOPATH)/bin/kine"
	chmod +x "$(shell go env GOPATH)/bin/kine"
	curl -fsSL "https://dl.k8s.io/$(KUBE_APISERVER_VERSION)/bin/linux/amd64/kube-apiserver" \
		-o "$(shell go env GOPATH)/bin/kube-apiserver"
	chmod +x "$(shell go env GOPATH)/bin/kube-apiserver"

##@ Helm

HELM_UNITTEST_VERSION := 1.0.3

.PHONY: helm-lint
helm-lint: ## Lint the chart.
	helm lint helm/mcp-capi

.PHONY: helm-test
helm-test: helm-lint helm-unittest ## Run every chart check (what the chart-test CI job runs).

.PHONY: helm-unittest
helm-unittest: helm-plugin-unittest ## Run the helm-unittest suites in helm/mcp-capi/tests/.
	helm unittest helm/mcp-capi

.PHONY: helm-plugin-unittest
helm-plugin-unittest:
	@helm plugin list | grep -q '^unittest' || helm plugin install https://github.com/helm-unittest/helm-unittest --version $(HELM_UNITTEST_VERSION)
