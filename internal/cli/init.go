package cli

import (
	"fmt"
	"os"

	"arbor/internal/config"

	"github.com/spf13/cobra"
)

func newInitCommand(opts *Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize an Arbor config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				if _, err := os.Stat(opts.ConfigPath); err == nil {
					return fmt.Errorf("config already exists at %s", opts.ConfigPath)
				}
			}
			cfg := config.StarterConfig()
			cfg.Path = opts.ConfigPath
			data, err := cfg.MarshalYAML()
			if err != nil {
				return err
			}
			if err := os.WriteFile(opts.ConfigPath, data, 0o644); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", opts.ConfigPath)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")

	return cmd
}
