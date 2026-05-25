package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"dolphin/internal/i18n"

	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	var userMode bool

	cmd := &cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdInstallUse),
		Short: i18n.TL(i18n.KeyCmdInstallShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(userMode)
		},
	}

	cmd.Flags().BoolVarP(&userMode, "user", "u", false,
		"install to user-local bin directory (~/.local/bin)")

	return cmd
}

func runInstall(userMode bool) error {
	current, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current binary: %w", err)
	}
	if real, err := filepath.EvalSymlinks(current); err == nil {
		current = real
	}
	currentAbs, _ := filepath.Abs(current)

	target := installTarget(userMode)
	targetAbs, _ := filepath.Abs(target)

	// Already installed at target
	if currentAbs == targetAbs {
		fmt.Fprintf(os.Stderr, "Already installed at %s\n", target)
		return nil
	}

	// Read current binary
	data, err := os.ReadFile(current)
	if err != nil {
		return fmt.Errorf("read current binary: %w", err)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	// Write binary
	if err := os.WriteFile(target, data, 0755); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Permission denied. Try:\n")
			fmt.Fprintf(os.Stderr, "  sudo %s install\n", filepath.Base(current))
			if !userMode {
				fmt.Fprintf(os.Stderr, "\nOr install to user-local directory:\n")
				fmt.Fprintf(os.Stderr, "  %s install --user\n", filepath.Base(current))
			}
			return nil
		}
		return fmt.Errorf("write binary to %s: %w", target, err)
	}

	fmt.Fprintf(os.Stderr, "Installed to %s\n", target)

	// Warn if target is not in PATH
	if runtime.GOOS != "windows" {
		path := os.Getenv("PATH")
		if !strings.Contains(path, targetDir) {
			fmt.Fprintf(os.Stderr, "Make sure %s is in your PATH\n", targetDir)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Make sure %s is in your PATH (%%PATH%%)\n", targetDir)
	}

	return nil
}

// installTarget returns the install path for the dolphin binary.
func installTarget(userMode bool) string {
	if userMode {
		return userBinPath()
	}
	return systemBinPath()
}

func userBinPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "bin", binaryName())
}

func systemBinPath() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(localAppData, "dolphin", "bin", binaryName())
	}
	// macOS and Linux
	return filepath.Join("/", "usr", "local", "bin", binaryName())
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "dolphin.exe"
	}
	return "dolphin"
}
