package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	completionShellBash       = "bash"
	completionShellZsh        = "zsh"
	completionShellFish       = "fish"
	completionShellPowerShell = "powershell"
)

type completionInstaller struct {
	scriptPath      string
	scriptFileMode  os.FileMode
	scriptGenerator func(io.Writer) error
	postInstall     func(home, scriptPath string) ([]string, error)
}

func newCompletionCommand() *cobra.Command {
	var forceStdout bool
	var installPath string
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate or install shell completion scripts",
		ValidArgs: []string{completionShellBash, completionShellZsh, completionShellFish, completionShellPowerShell},
		Args:      cobra.ExactValidArgs(1),
		Long: "Generate a shell completion script when writing to stdout, or install it for your shell when run in an interactive terminal.\n\n" +
			"Examples:\n" +
			"  arbor completion zsh\n" +
			"  arbor completion bash --stdout\n" +
			"  arbor completion fish --path ~/.config/fish/completions/arbor.fish",
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			if forceStdout || !writerLooksInteractive(cmd.OutOrStdout()) {
				return generateCompletion(cmd.Root(), shell, noDescriptions, cmd.OutOrStdout())
			}

			installer, err := buildCompletionInstaller(cmd.Root(), shell, installPath, noDescriptions)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(installer.scriptPath), 0o755); err != nil {
				return fmt.Errorf("create completion directory: %w", err)
			}

			scriptFile, err := os.OpenFile(installer.scriptPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, installer.scriptFileMode)
			if err != nil {
				return fmt.Errorf("open completion script: %w", err)
			}
			defer scriptFile.Close()

			if err := installer.scriptGenerator(scriptFile); err != nil {
				return err
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve home directory: %w", err)
			}

			followUps, err := installer.postInstall(home, installer.scriptPath)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "installed %s completion to %s\n", shell, installer.scriptPath); err != nil {
				return err
			}
			for _, line := range followUps {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&forceStdout, "stdout", false, "print the completion script to stdout instead of installing it")
	cmd.Flags().StringVar(&installPath, "path", "", "custom install path for the generated completion script")
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions")

	return cmd
}

func writerLooksInteractive(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(file.Fd()))
}

func generateCompletion(root *cobra.Command, shell string, noDescriptions bool, w io.Writer) error {
	switch shell {
	case completionShellBash:
		return root.GenBashCompletionV2(w, !noDescriptions)
	case completionShellZsh:
		if noDescriptions {
			return root.GenZshCompletionNoDesc(w)
		}
		return root.GenZshCompletion(w)
	case completionShellFish:
		return root.GenFishCompletion(w, !noDescriptions)
	case completionShellPowerShell:
		if noDescriptions {
			return root.GenPowerShellCompletion(w)
		}
		return root.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func buildCompletionInstaller(root *cobra.Command, shell, installPath string, noDescriptions bool) (completionInstaller, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return completionInstaller{}, fmt.Errorf("resolve home directory: %w", err)
	}

	scriptPath := installPath
	if scriptPath == "" {
		scriptPath = defaultCompletionPath(home, shell)
	}

	installer := completionInstaller{
		scriptPath:      scriptPath,
		scriptFileMode:  0o644,
		scriptGenerator: func(w io.Writer) error { return generateCompletion(root, shell, noDescriptions, w) },
		postInstall: func(string, string) ([]string, error) {
			return []string{"open a new shell or source your shell config to start using completions"}, nil
		},
	}

	switch shell {
	case completionShellZsh:
		installer.postInstall = installZshCompletion
	case completionShellBash:
		installer.postInstall = installBashCompletion
	case completionShellFish:
		installer.postInstall = installFishCompletion
	case completionShellPowerShell:
		installer.postInstall = installPowerShellCompletion
	default:
		return completionInstaller{}, fmt.Errorf("unsupported shell %q", shell)
	}

	return installer, nil
}

func defaultCompletionPath(home, shell string) string {
	switch shell {
	case completionShellZsh:
		return filepath.Join(home, ".zsh", "completions", "_arbor")
	case completionShellBash:
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "arbor")
	case completionShellFish:
		return filepath.Join(home, ".config", "fish", "completions", "arbor.fish")
	case completionShellPowerShell:
		return filepath.Join(home, ".config", "powershell", "completions", "arbor.ps1")
	default:
		return ""
	}
}

func installZshCompletion(home, scriptPath string) ([]string, error) {
	zshrc := filepath.Join(home, ".zshrc")
	completionsDir := filepath.Dir(scriptPath)
	block := strings.Join([]string{
		"# arbor completion",
		fmt.Sprintf(`if [[ -d %q ]]; then`, completionsDir),
		fmt.Sprintf(`  fpath=(%q $fpath)`, completionsDir),
		"fi",
	}, "\n") + "\n"

	if err := ensureFileContains(zshrc, block); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(zshrc)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", zshrc, err)
	}
	if !strings.Contains(string(content), "compinit") {
		compinitBlock := strings.Join([]string{
			"# enable zsh completion",
			"autoload -Uz compinit",
			"compinit",
		}, "\n") + "\n"
		if err := ensureFileContains(zshrc, compinitBlock); err != nil {
			return nil, err
		}
	}

	return []string{fmt.Sprintf("updated %s to load Arbor completions", zshrc)}, nil
}

func installBashCompletion(home, scriptPath string) ([]string, error) {
	bashrc := filepath.Join(home, ".bashrc")
	block := strings.Join([]string{
		"# arbor completion",
		fmt.Sprintf(`if [[ -r %q ]]; then`, scriptPath),
		fmt.Sprintf(`  source %q`, scriptPath),
		"fi",
	}, "\n") + "\n"

	if err := ensureFileContains(bashrc, block); err != nil {
		return nil, err
	}

	return []string{fmt.Sprintf("updated %s to source Arbor completions", bashrc)}, nil
}

func installFishCompletion(home, scriptPath string) ([]string, error) {
	if scriptPath == defaultCompletionPath(home, completionShellFish) {
		return []string{"fish loads completions from ~/.config/fish/completions automatically"}, nil
	}

	return []string{fmt.Sprintf("source %s from your fish config to use this custom install path", scriptPath)}, nil
}

func installPowerShellCompletion(home, scriptPath string) ([]string, error) {
	profile := filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	block := strings.Join([]string{
		"# arbor completion",
		fmt.Sprintf(`. %q`, scriptPath),
	}, "\n") + "\n"

	if err := ensureFileContains(profile, block); err != nil {
		return nil, err
	}

	return []string{fmt.Sprintf("updated %s to source Arbor completions", profile)}, nil
}

func ensureFileContains(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if strings.Contains(string(content), block) {
		return nil
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write newline to %s: %w", path, err)
		}
	}

	if _, err := file.WriteString(block); err != nil {
		return fmt.Errorf("append to %s: %w", path, err)
	}

	return nil
}
