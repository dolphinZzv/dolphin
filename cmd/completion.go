package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for dolphin commands.

Output the completion script for the specified shell.
Source the output to enable tab completion.

  bash:       source <(dolphin completion bash)
  zsh:        source <(dolphin completion zsh)
  fish:       dolphin completion fish | source
  powershell: dolphin completion powershell | Out-String | Invoke-Expression

To make it permanent (bash):
  dolphin completion bash > /etc/bash_completion.d/dolphin

To make it permanent (zsh):
  dolphin completion zsh > "${fpath[1]}/_dolphin"`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                 cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (use bash, zsh, fish, or powershell)", args[0])
			}
		},
	}
}
