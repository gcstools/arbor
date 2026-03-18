package cli

import (
	"context"
	"fmt"
	"os"

	"arbor/internal/gitutil"

	"github.com/spf13/cobra"
)

func newPullCommand(_ *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull the main worktree when it has no local changes.",
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			ctx := context.Background()
			repoState, err := gitutil.DiscoverRepo(ctx, cwd)
			if err != nil {
				return err
			}

			mainPath, err := gitutil.Runner{Dir: repoState.Root}.MainWorktreePath(ctx)
			if err != nil {
				return err
			}

			restore := func() error { return nil }
			if cwd != mainPath {
				if err := os.Chdir(mainPath); err != nil {
					return err
				}
				restore = func() error {
					if err := os.Chdir(cwd); err != nil {
						return fmt.Errorf("restore working directory: %w", err)
					}
					return nil
				}
			}
			defer func() {
				if err := restore(); err != nil && runErr == nil {
					runErr = err
				}
			}()

			runner := gitutil.Runner{Dir: mainPath}
			dirty, err := runner.IsDirty(ctx)
			if err != nil {
				return err
			}
			if dirty {
				_, runErr = fmt.Fprintf(cmd.OutOrStdout(), "skipped pull: main worktree has local changes at %s\n", mainPath)
				return runErr
			}

			if err := runner.Pull(ctx); err != nil {
				return err
			}
			_, runErr = fmt.Fprintf(cmd.OutOrStdout(), "pulled main worktree: %s\n", mainPath)
			return runErr
		},
	}

	return cmd
}
