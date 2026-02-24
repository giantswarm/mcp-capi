package integration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestMain(m *testing.M) {
	if err := harness.InitManager(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize test harness: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	harness.ShutdownManager()
	os.Exit(code)
}
