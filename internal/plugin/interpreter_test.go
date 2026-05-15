//go:build !windows

package plugin

import (
	"testing"
)

func TestShellInterpreterUnix(t *testing.T) {
	interp := shellInterpreter()
	if interp != "sh" {
		t.Errorf("shellInterpreter() = %q, want sh", interp)
	}
}
