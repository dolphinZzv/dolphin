//go:build windows

package plugin

import "strings"

// trimScriptExt removes the known script extension from a file name.
// On Windows, .ps1, .bat, .cmd, and .sh are recognized.
func trimScriptExt(name string) string {
	for _, ext := range []string{".ps1", ".bat", ".cmd", ".sh"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}
