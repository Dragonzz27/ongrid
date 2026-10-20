//go:build !windows

package runner

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureProcessGroup runs the child in its own process group and makes
// context cancellation kill the whole group, so a timed-out script's
// grandchildren (e.g. `cd x && terraform apply`) die with it instead of
// holding the output pipes open.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return cmd.Process.Kill()
		}
		return nil
	}
}
