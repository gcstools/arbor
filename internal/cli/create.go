package cli

import (
	"context"
	"fmt"
	"os"

	"arbor/internal/execute"
	"arbor/internal/planner"

	"github.com/spf13/cobra"
)

func newCreateCommand(opts *Options) *cobra.Command {
	var preset string
	var baseRef string
	var openApp string
	var branchTemplate string
	var pathTemplate string
	var nonInteractive bool
	var planOnly bool

	cmd := &cobra.Command{
		Use:   "create [name...]",
		Short: "Create worktrees and run setup actions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			plan, err := planner.BuildCreatePlan(context.Background(), planner.Inputs{
				CWD:            cwd,
				Names:          args,
				BaseRef:        baseRef,
				Preset:         preset,
				OpenApp:        openApp,
				BranchTemplate: branchTemplate,
				PathTemplate:   pathTemplate,
				NonInteractive: nonInteractive,
			}, cmd.InOrStdin(), opts.ConfigPath)
			if err != nil {
				return formatCreatePlanError(err)
			}
			if planOnly {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), planner.RenderSummary(plan))
				return err
			}

			summary, err := execute.Runner{
				Stdout: cmd.OutOrStdout(),
				Stderr: cmd.ErrOrStderr(),
			}.Apply(context.Background(), plan)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), renderExecutionSummary(summary))
			return err
		},
	}
	cmd.Flags().StringVar(&preset, "preset", "", "preset to apply")
	cmd.Flags().StringVar(&baseRef, "base", "", "base branch or commit")
	cmd.Flags().StringVar(&openApp, "open-app", "", "app executable to open the created worktree folder")
	cmd.Flags().StringVar(&branchTemplate, "branch-template", "", "branch naming template")
	cmd.Flags().StringVar(&pathTemplate, "path-template", "", "worktree path template")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts and require resolved inputs")
	cmd.Flags().BoolVar(&planOnly, "plan", false, "show the create plan without executing it")

	return cmd
}
