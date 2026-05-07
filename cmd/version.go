package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd creates the Cobra command for displaying the application version.
// The actual version information is typically managed by the root command or a global variable.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print the version number of %s", serviceName),
		Long:  `All software has versions. This is its.`,
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", serviceName, rootCmd.Version)
		},
	}
}
