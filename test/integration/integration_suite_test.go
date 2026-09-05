package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

// testNamespace is the default namespace used across integration tests.
const testNamespace = "test-clusters"

// kubeconfigSecretKey is the data key under which kubeconfig contents are stored
// in CAPI-managed Secrets (matches the convention used by cluster-api).
const kubeconfigSecretKey = "value"

// Helm's ownership marker, as carried by every workload cluster a Giant Swarm
// cluster chart renders on a management cluster. The GitOps guard refuses
// writes to such objects.
const (
	managedByLabel = "app.kubernetes.io/managed-by"
	managedByHelm  = "Helm"
)

func TestMain(m *testing.M) {
	// The harness needs kine and kube-apiserver. `make test` installs them
	// first (the install-test-binaries prerequisite), so the architect go-build
	// test_target runs the full suite. A bare `go test ./...` without them
	// (e.g. a quick local run) skips instead of failing.
	for _, bin := range []string{"kine", "kube-apiserver"} {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Fprintf(os.Stderr, "skipping integration tests: %s not found in PATH (run `make install-test-binaries`)\n", bin)
			os.Exit(0)
		}
	}
	if err := harness.InitManager(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize test harness: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	harness.ShutdownManager()
	os.Exit(code)
}
