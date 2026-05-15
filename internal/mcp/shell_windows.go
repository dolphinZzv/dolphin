//go:build windows

package mcp

import (
	"context"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	detectedShell string
	detectOnce    sync.Once
)

func detectShell() string {
	// Priority: PowerShell Core > Windows PowerShell > cmd.exe > Git Bash/WSL bash
	for _, name := range []string{"pwsh.exe", "powershell.exe", "cmd.exe", "bash.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return "cmd.exe"
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	detectOnce.Do(func() {
		detectedShell = detectShell()
	})

	switch filepath.Base(detectedShell) {
	case "pwsh.exe", "powershell.exe":
		return exec.CommandContext(ctx, detectedShell, "-NoProfile", "-Command", command)
	case "cmd.exe":
		return exec.CommandContext(ctx, detectedShell, "/C", command)
	default: // bash.exe or other
		return exec.CommandContext(ctx, detectedShell, "-c", command)
	}
}
