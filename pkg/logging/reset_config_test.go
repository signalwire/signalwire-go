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

// TestResetLoggingConfigurationCleanup restores logging config to the
// no-env-vars baseline AND verifies that ResetLoggingConfiguration is
// observable on globalLevel and suppressed: with no env vars set, the
// level must reset to LevelInfo and suppression must be off. This is
// also the cleanup hook for other tests in the package.
func TestResetLoggingConfigurationCleanup(t *testing.T) {
	t.Setenv("SIGNALWIRE_LOG_LEVEL", "")
	t.Setenv("SIGNALWIRE_LOG_MODE", "")

	// Pre-state: deliberately set both off-defaults so the reset is
	// observable. If reset does nothing, the post-state will still be
	// the "off" pre-state.
	SetGlobalLevel(LevelDebug)
	Suppress()

	ResetLoggingConfiguration()

	if got := GetGlobalLevel(); got != LevelInfo {
		t.Errorf("expected global level LevelInfo after reset with no env, got %v", got)
	}
	if IsSuppressed() {
		t.Errorf("expected suppression OFF after reset with no env vars")
	}
}
