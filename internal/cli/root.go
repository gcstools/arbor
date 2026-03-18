package cli

import (
	"fmt"
	"io"

	"arbor/internal/config"

	"github.com/spf13/cobra"
)

type Options struct {
	ConfigPath string
	NoColor    bool
}

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func NewRootCommand(streams IOStreams) *cobra.Command {
	opts := &Options{}

	rootCmd := &cobra.Command{
		Use:           "arbor",
		Short:         "Manage Git worktree setup workflows.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if opts.ConfigPath == "" {
				opts.ConfigPath = config.DefaultConfigPath
			}
			return nil
		},
	}

	rootCmd.SetOut(streams.Out)
	rootCmd.SetErr(streams.ErrOut)
	rootCmd.SetIn(streams.In)

	rootCmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", config.DefaultConfigPath, "path to Arbor config")
	rootCmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colorized output")

	rootCmd.AddCommand(
		newCreateCommand(opts),
		newInitCommand(opts),
		newDetectCommand(opts),
		newPullCommand(opts),
		newConfigCommand(opts),
		newVersionCommand(),
		newCompletionCommand(),
	)

	return rootCmd
}

func Execute(args []string, streams IOStreams) error {
	rootCmd := NewRootCommand(streams)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func renderPlaceholder(cmd *cobra.Command, message string, opts *Options, args []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\nconfig: %s\nargs: %v\n", message, opts.ConfigPath, args)
	return err
}
