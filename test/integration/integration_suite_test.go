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

func TestMain(m *testing.M) {
	// The harness needs kine and kube-apiserver (installed via
	// `make install-test-binaries`). Plain `go test ./...` runs without them
	// (e.g. the generated CircleCI go-build job), so skip instead of failing;
	// the Lint and Test workflow installs the binaries and keeps full coverage.
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
