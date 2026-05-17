//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBinary_Unix(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "dolphin")
	oldData := []byte("old-binary")
	newData := []byte("new-binary-data")

	if err := os.WriteFile(execPath, oldData, 0755); err != nil {
		t.Fatal(err)
	}

	if err := InstallBinary(newData, execPath); err != nil {
		t.Fatalf("InstallBinary: %v", err)
	}

	got, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newData) {
		t.Errorf("binary = %q, want %q", string(got), string(newData))
	}

	// Backup should be cleaned up.
	//nolint:govet
	if _, err := os.Stat(execPath + ".bak"); !os.IsNotExist(err) {
		t.Error("expected .bak file to be removed after successful install")
	}

	// Verify file mode is executable.
	info, err := os.Stat(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("binary is not executable")
	}
}

func TestInstallBinary_Rollback(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "dolphin")
	oldData := []byte("old-binary")
	newData := []byte("new-data")

	if err := os.WriteFile(execPath, oldData, 0755); err != nil {
		t.Fatal(err)
	}

	if err := InstallBinary(newData, execPath); err != nil {
		t.Fatalf("InstallBinary: %v", err)
	}

	got, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newData) {
		t.Errorf("binary = %q, want %q", string(got), string(newData))
	}
}
