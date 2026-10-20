//go:build windows

package runner

import "os/exec"

func configureProcessGroup(*exec.Cmd) {}
