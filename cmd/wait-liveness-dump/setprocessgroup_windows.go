// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build windows

package main

import "os/exec"

// setProcessGroup is a no-op on Windows: syscall.SysProcAttr has no Setpgid field
// there (referencing it would not compile), and the child is killed directly on
// exit, so the process-group isolation is unnecessary.
func setProcessGroup(cmd *exec.Cmd) {}
