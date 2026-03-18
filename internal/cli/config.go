package cli

import (
	"fmt"

	"arbor/internal/config"

	"github.com/spf13/cobra"
)

func newConfigCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or validate Arbor config.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(opts.ConfigPath)
			if err != nil {
				return err
			}
			data, err := cfg.MarshalYAML()
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}

	cmd.AddCommand(newConfigValidateCommand(opts))
	return cmd
}

func newConfigValidateCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the Arbor config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(opts.ConfigPath)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "config valid: %s\npresets: %d\n", cfg.Path, len(cfg.Presets))
			return err
		},
	}
}
