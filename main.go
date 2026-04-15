// Package main is the entry point for the mcp-capi binary.
package main

import (
	"github.com/giantswarm/mcp-capi/cmd"
)

// version will be set by goreleaser during build
var version = "dev"

func main() {
	// Set the version from build-time variable
	cmd.SetVersion(version)

	// Execute the root command
	cmd.Execute()
}
