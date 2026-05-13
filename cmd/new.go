package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Start a fresh dolphin session from a clean state",
		Long: `Cleans all dolphin runtime data and state, then starts a brand new
dolphin agent session.

Removed:
  - All runtime data (sessions, diary, logs, workspaces, crontab)
  - SSH auto-generated password
  - Cached tool manifests
  - Downloaded skills and commands
  - SYSTEM.md (system prompt)
  - /etc/dolphin/ system-level data
  - First-run marker

Config files (config.yaml) are preserved.`,
		RunE: runNew,
	}

	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	targets := cleanupTargets() // never removes config files

	// Show what will be removed
	fmt.Fprintln(os.Stderr, "Starting a fresh dolphin session. The following will be removed:")
	listTargets(targets)

	// Confirm
	if !force {
		if !confirmRemoval("new") {
			return nil
		}
	}

	removed, errors := doRemove(targets)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Cleanup complete: %d items removed", removed)
	if errors > 0 {
		fmt.Fprintf(os.Stderr, ", %d errors", errors)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)

	// Reset cfgFile so config.Load() doesn't try to load a stale -c path
	cfgFile = ""

	return runAgent(cmd, args)
}
