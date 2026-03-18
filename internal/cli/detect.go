package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"arbor/internal/config"
	"arbor/internal/detect"
	"arbor/internal/gitutil"

	"github.com/spf13/cobra"
)

func newDetectCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Preview detected env files and setup commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoState, err := gitutil.DiscoverRepo(context.Background(), cwd)
			if err != nil {
				return err
			}
			cfg, err := config.LoadOptional(filepathJoin(repoState.Root, opts.ConfigPath))
			if err != nil {
				return err
			}
			result, err := detect.Scan(repoState.Root, cfg)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "repo: %s\n", result.RepoRoot)
			fmt.Fprintln(cmd.OutOrStdout(), "env files:")
			if len(result.EnvFiles) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "  none")
			}
			for _, env := range result.EnvFiles {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s -> %s (%s, source=%s)\n", env.SourcePath, env.TargetPath, env.DefaultAction, env.Source)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "commands:")
			if len(result.Commands) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "  none")
			}
			for _, command := range result.Commands {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s [%s] source=%s\n", command.Command, command.Scope, command.Source)
			}
			if len(result.Warnings) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "warnings: %s\n", strings.Join(result.Warnings, ", "))
			}
			return nil
		},
	}

	return cmd
}
