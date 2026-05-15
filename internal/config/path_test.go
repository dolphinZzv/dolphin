package config

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultSessionDir(t *testing.T) {
	dir := defaultSessionDir()
	if dir == "" {
		t.Fatal("defaultSessionDir() returned empty")
	}
	if runtime.GOOS != "windows" && dir != "/tmp/dolphin" {
		t.Errorf("defaultSessionDir() = %q, want /tmp/dolphin", dir)
	}
}

func TestDefaultSystemConfigDir(t *testing.T) {
	dir := defaultSystemConfigDir()
	if dir == "" {
		t.Fatal("defaultSystemConfigDir() returned empty")
	}
	if runtime.GOOS != "windows" && dir != "/etc/dolphin" {
		t.Errorf("defaultSystemConfigDir() = %q, want /etc/dolphin", dir)
	}
}

func TestSessionDirUsesDefaultWhenEmpty(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Session.Dir == "" {
		t.Error("Session.Dir should not be empty after Load()")
	}
}

func TestDefaultConfigSessionDir(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Session.Dir == "" {
		t.Error("DefaultConfig().Session.Dir should not be empty")
	}
}

func TestSystemConfigDirVar(t *testing.T) {
	if SystemConfigDir == "" {
		t.Error("SystemConfigDir should not be empty")
	}
}

func TestHomeDirFallback(t *testing.T) {
	// Simulate os.UserHomeDir() failure by setting a fake HOME that doesn't exist
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/nonexistent_home_12345")
	defer os.Setenv("HOME", oldHome)

	// When UserHomeDir fails and the fallback is used, line 434 sets hd = os.TempDir()
	// Then it tries to read the SSH password file from <hd>/.dolphin/ssh_password
	// which won't exist, so it will generate a new one. This should not crash.
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error with bad HOME: %v", err)
	}
	if cfg.Session.Dir == "" {
		t.Error("Session.Dir should not be empty even with bad HOME")
	}
}
