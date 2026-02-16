package integration_test

import (
	"os"
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestMain(m *testing.M) {
	harness.InitManager()
	code := m.Run()
	harness.ShutdownManager()
	os.Exit(code)
}
