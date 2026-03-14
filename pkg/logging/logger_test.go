package logging

import (
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"off", LevelOff},
		{"none", LevelOff},
		{"silent", LevelOff},
		{"unknown", LevelInfo},
		{"", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSetGetGlobalLevel(t *testing.T) {
	original := GetGlobalLevel()
	defer SetGlobalLevel(original)

	SetGlobalLevel(LevelDebug)
	if got := GetGlobalLevel(); got != LevelDebug {
		t.Errorf("GetGlobalLevel() = %v, want %v", got, LevelDebug)
	}

	SetGlobalLevel(LevelError)
	if got := GetGlobalLevel(); got != LevelError {
		t.Errorf("GetGlobalLevel() = %v, want %v", got, LevelError)
	}
}

func TestSuppression(t *testing.T) {
	original := IsSuppressed()
	defer func() {
		if original {
			Suppress()
		} else {
			Unsuppress()
		}
	}()

	Suppress()
	if !IsSuppressed() {
		t.Error("expected logging to be suppressed")
	}

	Unsuppress()
	if IsSuppressed() {
		t.Error("expected logging to be unsuppressed")
	}
}

func TestLoggerCreation(t *testing.T) {
	l := New("test-component")
	if l.name != "test-component" {
		t.Errorf("logger name = %q, want %q", l.name, "test-component")
	}
	if l.logger == nil {
		t.Error("logger internal logger is nil")
	}
}

func TestLoggerShouldLog(t *testing.T) {
	original := GetGlobalLevel()
	defer SetGlobalLevel(original)

	l := New("test")

	SetGlobalLevel(LevelInfo)
	if l.shouldLog(LevelDebug) {
		t.Error("debug should not log at info level")
	}
	if !l.shouldLog(LevelInfo) {
		t.Error("info should log at info level")
	}
	if !l.shouldLog(LevelWarn) {
		t.Error("warn should log at info level")
	}
	if !l.shouldLog(LevelError) {
		t.Error("error should log at info level")
	}
}

func TestLoggerSuppressed(t *testing.T) {
	original := IsSuppressed()
	defer func() {
		if original {
			Suppress()
		} else {
			Unsuppress()
		}
	}()

	l := New("test")
	Suppress()

	if l.shouldLog(LevelError) {
		t.Error("no messages should log when suppressed")
	}
}

func TestLoggerMethodsDoNotPanic(t *testing.T) {
	// Ensure logging methods don't panic even when suppressed
	Suppress()
	defer Unsuppress()

	l := New("test")
	l.Debug("test %s", "debug")
	l.Info("test %s", "info")
	l.Warn("test %s", "warn")
	l.Error("test %s", "error")
}
