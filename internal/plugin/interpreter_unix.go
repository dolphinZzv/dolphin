//go:build !windows

package plugin

func shellInterpreter() string {
	return "sh"
}
