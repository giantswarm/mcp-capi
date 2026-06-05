KINE_VERSION := v0.14.12
KUBE_APISERVER_VERSION := v1.35.2

.PHONY: install-test-binaries
install-test-binaries: ## Install kine and kube-apiserver needed by integration tests (Linux amd64).
	curl -fsSL "https://github.com/k3s-io/kine/releases/download/$(KINE_VERSION)/kine-amd64" \
		-o "$(shell go env GOPATH)/bin/kine"
	chmod +x "$(shell go env GOPATH)/bin/kine"
	curl -fsSL "https://dl.k8s.io/$(KUBE_APISERVER_VERSION)/bin/linux/amd64/kube-apiserver" \
		-o "$(shell go env GOPATH)/bin/kube-apiserver"
	chmod +x "$(shell go env GOPATH)/bin/kube-apiserver"
