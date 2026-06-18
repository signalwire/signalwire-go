// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build unix

package mocktest

import (
	"os/exec"
	"syscall"
)

// setProcessGroup runs the spawned mock-server process in its own process group
// (Setpgid: true) so signals to the test binary do not cascade to the child and
// the child stays detached (keeps the testing framework's pipe-drain logic from
// blocking on it). Unix-only: the Setpgid field does not exist on Windows.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
