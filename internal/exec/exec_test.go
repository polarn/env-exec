package exec

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const trapScript = `
trap 'exit 42' TERM
trap 'exit 43' INT
touch "$1"
i=0
while [ ! -e "$2" ] && [ $i -lt 200 ]; do sleep 0.05; i=$((i+1)); done
exit 5
`

type result struct {
	code int
	err  error
}

func startTrapScript(t *testing.T) (release func(), results <-chan result) {
	t.Helper()

	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	done := filepath.Join(dir, "done")
	release = func() { os.WriteFile(done, nil, 0600) }
	t.Cleanup(release)

	ch := make(chan result, 1)
	go func() {
		code, err := Run([]string{"sh", "-c", trapScript, "sh", ready, done})
		ch <- result{code, err}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			return release, ch
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
		t.Fatal("Run did not return")
		return result{}
	}
}

func setCtrlCReachesChild(t *testing.T, reaches bool) {
	t.Helper()
	original := ctrlCReachesChild
	ctrlCReachesChild = func() bool { return reaches }
	t.Cleanup(func() { ctrlCReachesChild = original })
}

func TestRun_ExitCode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"success", []string{"sh", "-c", "exit 0"}, 0},
		{"failure", []string{"sh", "-c", "exit 3"}, 3},
		{"killed by signal", []string{"sh", "-c", "kill -TERM $$"}, 128 + int(syscall.SIGTERM)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := Run(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.want {
				t.Errorf("want exit code %d, got %d", tt.want, code)
			}
		})
	}
}

func TestRun_CommandNotFound(t *testing.T) {
	_, err := Run([]string{"env-exec-no-such-command"})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to run command") {
		t.Errorf("want 'failed to run command', got %q", err.Error())
	}
}

func TestRun_ForwardsSignals(t *testing.T) {
	tests := []struct {
		name         string
		signal       syscall.Signal
		ctrlCReaches bool
		want         int
	}{
		{"SIGTERM", syscall.SIGTERM, false, 42},
		{"SIGTERM with terminal", syscall.SIGTERM, true, 42},
		{"SIGINT without terminal", syscall.SIGINT, false, 43},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCtrlCReachesChild(t, tt.ctrlCReaches)
			_, results := startTrapScript(t)

			if err := syscall.Kill(os.Getpid(), tt.signal); err != nil {
				t.Fatal(err)
			}

			r := await(t, results)
			if r.err != nil {
				t.Fatalf("unexpected error: %v", r.err)
			}
			if r.code != tt.want {
				t.Errorf("want exit code %d, got %d", tt.want, r.code)
			}
		})
	}
}

func TestRun_DoesNotRepeatTerminalInterrupt(t *testing.T) {
	setCtrlCReachesChild(t, true)
	release, results := startTrapScript(t)

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	release()

	r := await(t, results)
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if r.code != 5 {
		t.Errorf("want exit code 5 (SIGINT not forwarded), got %d", r.code)
	}
}
