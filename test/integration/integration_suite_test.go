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
	// The integration suite spins up an envtest control plane, which needs the
	// kube-apiserver and kine binaries on PATH (installed by .github/workflows/
	// ci.yaml). Environments that don't provide them -- e.g. the architect
	// go-build job that runs `go test ./...` -- skip the suite rather than fail,
	// so the default test run stays green wherever the binaries are absent.
	for _, bin := range []string{"kube-apiserver", "kine"} {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Fprintf(os.Stderr, "skipping integration tests: %q not found on PATH (envtest binaries required)\n", bin)
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
