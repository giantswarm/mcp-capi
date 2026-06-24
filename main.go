package main

import (
	"github.com/giantswarm/mcp-capi/cmd"
)

// version is set at build time via -X ldflags by architect-orb's go-build job
var version = "dev"

func main() {
	// Set the version from build-time variable
	cmd.SetVersion(version)

	// Execute the root command
	cmd.Execute()
}
