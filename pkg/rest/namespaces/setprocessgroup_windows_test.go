// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build windows

package namespaces_test

import "os/exec"

// tlsSetProcessGroup is a no-op on Windows: syscall.SysProcAttr has no Setpgid
// field there (referencing it would not compile), and Windows terminates the
// child directly, so process-group isolation is unnecessary.
func tlsSetProcessGroup(cmd *exec.Cmd) {}
