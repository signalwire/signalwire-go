// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build windows

package mocktest

import "os/exec"

// setProcessGroup is a no-op on Windows. The Unix process-group isolation
// (Setpgid) has no Windows equivalent here, and syscall.SysProcAttr has no
// Setpgid field on Windows (referencing it would not compile). Windows
// terminates the child process directly, so the isolation is unnecessary.
func setProcessGroup(cmd *exec.Cmd) {}
