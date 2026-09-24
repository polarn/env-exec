package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("want %s empty, got %v", dir, entries)
	}
}

func TestWriteFiles(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	key := "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n"
	envVars := map[string]string{"KEY_PATH": key, "PLAIN": "plain"}

	remove, err := WriteFiles(envVars, []string{"KEY_PATH", "MISSING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := envVars["KEY_PATH"]
	dir := filepath.Dir(path)
	if filepath.Dir(dir) != runtimeDir || !strings.HasPrefix(filepath.Base(dir), "env-exec-") {
		t.Errorf("want file in %s/env-exec-*, got %s", runtimeDir, path)
	}
	if filepath.Base(path) != "KEY_PATH" {
		t.Errorf("want file named after the variable, got %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("want file mode 0600, got %o", mode)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mode := dirInfo.Mode().Perm(); mode != 0700 {
		t.Errorf("want directory mode 0700, got %o", mode)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != key {
		t.Errorf("want contents %q, got %q", key, got)
	}

	if envVars["PLAIN"] != "plain" {
		t.Errorf("want PLAIN untouched, got %q", envVars["PLAIN"])
	}
	if _, ok := envVars["MISSING"]; ok {
		t.Error("want MISSING left unset")
	}

	if err := remove(); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	assertEmptyDir(t, runtimeDir)
}

func TestWriteFiles_FallsBackToTempDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", tmpDir)

	envVars := map[string]string{"KEY_PATH": "secret"}
	remove, err := WriteFiles(envVars, []string{"KEY_PATH"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer remove()

	if dir := filepath.Dir(filepath.Dir(envVars["KEY_PATH"])); dir != tmpDir {
		t.Errorf("want file under %s, got %s", tmpDir, envVars["KEY_PATH"])
	}
}

func TestWriteFiles_NoNames(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	envVars := map[string]string{"PLAIN": "plain"}
	remove, err := WriteFiles(envVars, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := remove(); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}

	assertEmptyDir(t, runtimeDir)
	if envVars["PLAIN"] != "plain" {
		t.Errorf("want PLAIN untouched, got %q", envVars["PLAIN"])
	}
}

func TestWriteFiles_RejectsEscapingName(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	envVars := map[string]string{"OK": "fine", "../escape": "secret"}
	_, err := WriteFiles(envVars, []string{"OK", "../escape"})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "'../escape'") {
		t.Errorf("want error naming the variable, got %q", err.Error())
	}

	assertEmptyDir(t, runtimeDir)
	if envVars["OK"] != "fine" {
		t.Errorf("want envVars untouched on error, got OK=%q", envVars["OK"])
	}
}
