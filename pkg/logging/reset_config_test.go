// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package logging

import (
	"testing"
)

// Tests for ResetLoggingConfiguration added in this PR. Mirrors Python
// reset_logging_configuration() semantics: re-read SIGNALWIRE_LOG_LEVEL
// and SIGNALWIRE_LOG_MODE from the environment and replace the current
// global level + suppression state.

func TestResetLoggingConfigurationRestoresInfoDefault(t *testing.T) {
	// Ensure env is clean.
	t.Setenv("SIGNALWIRE_LOG_LEVEL", "")
	t.Setenv("SIGNALWIRE_LOG_MODE", "")

	// Mutate global state via SetGlobalLevel + Suppress...
	SetGlobalLevel(LevelDebug)
	Suppress()
	if GetGlobalLevel() != LevelDebug {
		t.Fatalf("precondition: expected LevelDebug after SetGlobalLevel")
	}
	if !IsSuppressed() {
		t.Fatalf("precondition: expected suppressed=true after Suppress()")
	}

	// Reset should restore Info + unsuppressed from an env-clean environment.
	ResetLoggingConfiguration()

	if GetGlobalLevel() != LevelInfo {
		t.Errorf("after reset: level = %v, want LevelInfo", GetGlobalLevel())
	}
	if IsSuppressed() {
		t.Errorf("after reset: suppressed should be false with clean env")
	}
}

func TestResetLoggingConfigurationHonorsLogLevelEnv(t *testing.T) {
	t.Setenv("SIGNALWIRE_LOG_LEVEL", "error")
	t.Setenv("SIGNALWIRE_LOG_MODE", "")

	SetGlobalLevel(LevelInfo)
	ResetLoggingConfiguration()

	if GetGlobalLevel() != LevelError {
		t.Errorf("expected reset to pick up SIGNALWIRE_LOG_LEVEL=error, got %v", GetGlobalLevel())
	}
}

func TestResetLoggingConfigurationHonorsLogModeOff(t *testing.T) {
	t.Setenv("SIGNALWIRE_LOG_LEVEL", "")
	t.Setenv("SIGNALWIRE_LOG_MODE", "off")

	Unsuppress()
	ResetLoggingConfiguration()

	if !IsSuppressed() {
		t.Errorf("expected reset to set suppressed=true when SIGNALWIRE_LOG_MODE=off")
	}
}

// Restore to clean state so other tests in the package start fresh.
func TestResetLoggingConfigurationCleanup(t *testing.T) {
	t.Setenv("SIGNALWIRE_LOG_LEVEL", "")
	t.Setenv("SIGNALWIRE_LOG_MODE", "")
	ResetLoggingConfiguration()
}
