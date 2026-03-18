package cli

import (
	"fmt"

	"arbor/internal/version"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Arbor version information.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"arbor %s\ncommit: %s\nbuilt: %s\n",
				version.Number,
				version.Commit,
				version.Date,
			)
			return err
		},
	}
}
