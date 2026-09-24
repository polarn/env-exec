package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const fileVarConfig = `env:
  - name: KEY_PATH
    asFile: true
    value: |
      -----BEGIN PRIVATE KEY-----
      abc
      -----END PRIVATE KEY-----
  - name: PLAIN
    value: plain
`

const fileVarScript = `
printf %%s "$KEY_PATH" > "$1.tmp"
mv "$1.tmp" "$1"
i=0
while [ ! -e "$2" ] && [ $i -lt 200 ]; do sleep 0.05; i=$((i+1)); done
exit %d
`

type result struct {
	code int
	err  error
}

func setup(t *testing.T, content string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "env-exec.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENV_EXEC_YAML", configPath)
	t.Setenv("KEY_PATH", "")
	t.Setenv("PLAIN", "")

	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	return runtimeDir
}

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

func startFileVarScript(t *testing.T, exitCode int) (path string, release func(), results <-chan result) {
	t.Helper()

	dir := t.TempDir()
	pathFile := filepath.Join(dir, "path")
	done := filepath.Join(dir, "done")
	release = func() { os.WriteFile(done, nil, 0600) }
	t.Cleanup(release)

	ch := make(chan result, 1)
	go func() {
		code, err := run([]string{"sh", "-c", fmt.Sprintf(fileVarScript, exitCode), "sh", pathFile, done})
		ch <- result{code, err}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if data, err := os.ReadFile(pathFile); err == nil {
			return string(data), release, ch
		}
		if time.Now().After(deadline) {
			t.Fatal("command did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func await(t *testing.T, results <-chan result) result {
	t.Helper()
	select {
	case r := <-results:
		return r
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return")
		return result{}
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRun_FileVar(t *testing.T) {
	for _, exitCode := range []int{0, 3} {
		t.Run(fmt.Sprintf("exit %d", exitCode), func(t *testing.T) {
			runtimeDir := setup(t, fileVarConfig)
			path, release, results := startFileVarScript(t, exitCode)

			if dir := filepath.Dir(filepath.Dir(path)); dir != runtimeDir {
				t.Errorf("want file under %s, got %s", runtimeDir, path)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if mode := info.Mode().Perm(); mode != 0600 {
				t.Errorf("want file mode 0600, got %o", mode)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n"
			if string(got) != want {
				t.Errorf("want contents %q, got %q", want, got)
			}

			release()
			r := await(t, results)
			if r.err != nil {
				t.Fatalf("unexpected error: %v", r.err)
			}
			if r.code != exitCode {
				t.Errorf("want exit code %d, got %d", exitCode, r.code)
			}
			if os.Getenv("PLAIN") != "plain" {
				t.Errorf("want PLAIN=plain, got %q", os.Getenv("PLAIN"))
			}
			assertEmptyDir(t, runtimeDir)
		})
	}
}

func TestRun_FileVarRemovedAfterInterrupt(t *testing.T) {
	runtimeDir := setup(t, fileVarConfig)
	_, release, results := startFileVarScript(t, 0)

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	release()

	if r := await(t, results); r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	assertEmptyDir(t, runtimeDir)
}

func TestRun_FileVarRejectedInExportMode(t *testing.T) {
	runtimeDir := setup(t, fileVarConfig)

	var err error
	output := captureStdout(t, func() { _, err = run(nil) })
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "'KEY_PATH': asFile needs a command to run") {
		t.Errorf("want asFile export mode error, got %q", err.Error())
	}
	if output != "" {
		t.Errorf("want no output, got %q", output)
	}
	assertEmptyDir(t, runtimeDir)
}

func TestRun_FileVarDryRun(t *testing.T) {
	for _, args := range [][]string{{"--dry-run"}, {"--dry-run", "true"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			runtimeDir := setup(t, fileVarConfig)

			var code int
			var err error
			output := captureStdout(t, func() { code, err = run(args) })
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != 0 {
				t.Errorf("want exit code 0, got %d", code)
			}
			for _, want := range []string{"export KEY_PATH='<file>'\n", "export PLAIN='plain'\n"} {
				if !strings.Contains(output, want) {
					t.Errorf("want output to contain %q, got %q", want, output)
				}
			}
			if strings.Contains(output, "PRIVATE KEY") {
				t.Errorf("want file contents hidden, got %q", output)
			}
			assertEmptyDir(t, runtimeDir)
		})
	}
}
