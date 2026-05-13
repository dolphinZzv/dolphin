package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dolphin/internal/config"

	"github.com/spf13/cobra"
)

func NewResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset dolphin to a clean state",
		Long: `Removes all runtime data, auto-generated files, and the first-run marker
so the next startup feels like the first time.

Runtime data removed:
  - Sessions, diary, logs, workspaces, crontab
  - SSH auto-generated password
  - Cached tool manifests
  - Downloaded skills and commands
  - SYSTEM.md (system prompt)
  - /etc/dolphin/ system-level config and data
  - First-run marker (setup wizard will show on next start)

Config files are preserved by default. Use --all to also remove user and project configs.`,
		RunE: runReset,
	}

	cmd.Flags().Bool("all", false, "also remove user and project config files")
	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	cmd.Flags().StringP("config", "c", "", "path to config file")

	return cmd
}

func runReset(cmd *cobra.Command, args []string) error {
	removeAll, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

	targets := cleanupTargets(removeAll)

	// Show what will be removed
	fmt.Fprintln(os.Stderr, "The following will be removed:")
	listTargets(targets)

	// Confirm
	if !force {
		if !confirmRemoval("reset") {
			return nil
		}
	}

	removed, errors := doRemove(targets)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Reset complete: %d items removed", removed)
	if errors > 0 {
		fmt.Fprintf(os.Stderr, ", %d errors", errors)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "\nThe first-run marker has been reset.")
	fmt.Fprintln(os.Stderr, "Run 'dolphin' to go through the initial setup wizard again.")

	return nil
}

// cleanupTargets builds the list of paths to remove for a dolphin reset.
func cleanupTargets(removeConfigs bool) []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	userDolphinDir := filepath.Join(homeDir, config.UserConfigDir)
	projectDolphinDir := config.ProjectConfigDir

	targets := []string{
		config.FirstRunMarker(),
		filepath.Join(userDolphinDir, "ssh_password"),
		filepath.Join(userDolphinDir, "SYSTEM.md"),
		filepath.Join(userDolphinDir, "cache"),
		filepath.Join(userDolphinDir, "skills"),
		filepath.Join(userDolphinDir, "commands"),
		filepath.Join(userDolphinDir, "plugins"),
		filepath.Join(projectDolphinDir, "sessions"),
		filepath.Join(projectDolphinDir, "diary"),
		filepath.Join(projectDolphinDir, "workspaces"),
		filepath.Join(projectDolphinDir, "logs"),
		filepath.Join(projectDolphinDir, "CRONTAB.md"),
		filepath.Join(projectDolphinDir, "skills"),
		filepath.Join(projectDolphinDir, "commands"),
		config.SystemConfigDir,
	}

	if removeConfigs {
		targets = append(targets,
			filepath.Join(userDolphinDir, "config.yaml"),
			filepath.Join(projectDolphinDir, "config.yaml"),
		)
	}

	return targets
}

// listTargets prints each target with its type (directory or file).
func listTargets(targets []string) {
	for _, t := range targets {
		info, err := os.Stat(t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  - %s (not found, skipped)\n", t)
			continue
		}
		if info.IsDir() {
			fmt.Fprintf(os.Stderr, "  - %s/ (directory)\n", t)
		} else {
			fmt.Fprintf(os.Stderr, "  - %s\n", t)
		}
	}
}

// confirmRemoval asks the user for confirmation. Returns true if confirmed.
func confirmRemoval(action string) bool {
	fmt.Fprintf(os.Stderr, "\nAre you sure? This action cannot be undone. [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		actionLabel := strings.ToUpper(action[:1]) + action[1:]
		fmt.Fprintf(os.Stderr, "%s cancelled.\n", actionLabel)
		return false
	}
	return true
}

// doRemove removes all given targets and returns counts.
func doRemove(targets []string) (removed, errors int) {
	for _, t := range targets {
		if err := os.RemoveAll(t); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to remove %s: %v\n", t, err)
			errors++
		} else {
			removed++
		}
	}
	return
}
