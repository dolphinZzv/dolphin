//go:build !windows

package config

func defaultSessionDir() string {
	return "/tmp/dolphin"
}

func defaultSystemConfigDir() string {
	return "/etc/dolphin"
}
