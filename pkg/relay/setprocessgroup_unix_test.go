// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build unix

package relay_test

import (
	"os/exec"
	"syscall"
)

// tlsSetProcessGroup runs the spawned mock under its own process group
// (Setpgid: true) on Unix so signals to the test binary don't cascade to it.
// Unix-only: the Setpgid field does not exist on Windows.
func tlsSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
