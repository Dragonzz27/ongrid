//go:build !windows

package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestShellRunnerTimeoutKillsDescendants(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	res, err := NewShellRunner().Run(context.Background(), Spec{
		Script:  "sleep 30 & echo $! > child.pid; wait",
		Workdir: dir,
		Timeout: 200 * time.Millisecond,
	})
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("want timeout error, got %v", err)
	}
	if res.ExitCode != -1 {
		t.Fatalf("exit = %d, want -1 on timeout", res.ExitCode)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Run returned after %s, want shortly after the 200ms timeout", elapsed)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "child.pid"))
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("parse child pid %q: %v", raw, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("background child %d still running after the timeout", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestShellRunnerDoesNotWaitForBackgroundOutput(t *testing.T) {
	start := time.Now()
	res, err := NewShellRunner().Run(context.Background(), Spec{Script: "sleep 30 & echo $!"})
	elapsed := time.Since(start)
	if pid, perr := strconv.Atoi(strings.TrimSpace(res.Stdout)); perr == nil {
		t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	}
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit = %d, want 0", res.ExitCode)
	}
	if strings.TrimSpace(res.Stdout) == "" {
		t.Fatal("stdout written before the shell exited was lost")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Run returned after %s, want it not to wait for the background child", elapsed)
	}
}
