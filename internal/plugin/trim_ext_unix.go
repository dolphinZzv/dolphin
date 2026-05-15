//go:build !windows

package plugin

import "strings"

// trimScriptExt removes the known script extension from a file name.
// On Unix, only .sh is recognized.
func trimScriptExt(name string) string {
	return strings.TrimSuffix(name, ".sh")
}
