//go:build windows

package plugin

import (
	"os/exec"
	"sync"
)

var (
	detectedInterpreter string
	interpreterOnce     sync.Once
)

func detectInterpreter() string {
	for _, name := range []string{"pwsh.exe", "powershell.exe", "cmd.exe", "bash.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return "cmd.exe"
}

func shellInterpreter() string {
	interpreterOnce.Do(func() {
		detectedInterpreter = detectInterpreter()
	})
	return detectedInterpreter
}
