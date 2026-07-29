package logging

import (
	"bytes"
	"os"
	"strings"
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

// TestLogLevelEnumOrString proves each typed LogLevel constant resolves through
// ParseLevel to the IDENTICAL Level as its bare-string equivalent — so the
// typed const and the plain string (which the Python reference uses) are
// behaviorally interchangeable. Real behavior — the live ParseLevel switch.
func TestLogLevelEnumOrString(t *testing.T) {
	cases := []struct {
		name LogLevel
		str  string
		want Level
	}{
		{LevelNameDebug, "debug", LevelDebug},
		{LevelNameInfo, "info", LevelInfo},
		{LevelNameWarn, "warn", LevelWarn},
		{LevelNameWarning, "warning", LevelWarn},
		{LevelNameError, "error", LevelError},
		{LevelNameOff, "off", LevelOff},
	}
	for _, c := range cases {
		// The defined-string constant's value is the canonical level name.
		if string(c.name) != c.str {
			t.Errorf("LogLevel const = %q, want %q", string(c.name), c.str)
		}
		// Typed const and bare string resolve to the same Level.
		gotConst := ParseLevel(string(c.name))
		gotStr := ParseLevel(c.str)
		if gotConst != c.want {
			t.Errorf("ParseLevel(%q const) = %v, want %v", c.name, gotConst, c.want)
		}
		if gotConst != gotStr {
			t.Errorf("typed const %q (%v) and string %q (%v) resolved to different Levels",
				c.name, gotConst, c.str, gotStr)
		}
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

func TestLoggerMethods_SuppressedSilencesAllLevels(t *testing.T) {
	// When the global Suppress() flag is on, every log method must drop
	// its message — no stderr output, no panic, no goroutine activity.
	// Capture stderr while invoking each method and assert it's empty.
	Suppress()
	defer Unsuppress()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	l := New("test")
	l.Debug("debug message %s", "d")
	l.Info("info message %s", "i")
	l.Warn("warn message %s", "w")
	l.Error("error message %s", "e")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	got := buf.String()
	for _, msg := range []string{"debug message", "info message", "warn message", "error message"} {
		if strings.Contains(got, msg) {
			t.Errorf("suppressed logger leaked %q to stderr; stderr=%q", msg, got)
		}
	}
}

// TestStripControlChars covers the reference's public event-map contract:
// string values are scrubbed, non-strings pass through untouched (the
// reference's `isinstance(value, str)` guard).
func TestStripControlChars(t *testing.T) {
	event := map[string]any{
		"event": "hello\x00world",
		"field": "a\x07b\x1fc",
		"n":     42,
	}

	out := StripControlChars(event)

	if out["event"] != "helloworld" {
		t.Errorf("event = %q, want %q", out["event"], "helloworld")
	}
	if out["field"] != "abc" {
		t.Errorf("field = %q, want %q", out["field"], "abc")
	}
	if out["n"] != 42 {
		t.Errorf("n = %v, want 42 (non-strings must pass through)", out["n"])
	}
}

// TestLogOutputHasControlCharsStripped proves the scrub is ON THE EMISSION PATH,
// not merely available. Seven ports in this fleet shipped strip_control_chars
// public and correct with ZERO call sites, so it protected nothing; go shipped
// without the function at all while PORT_OMISSIONS.md claimed the package
// "sanitises inline". The only assertion that can tell the difference is one that
// reads what the logger ACTUALLY wrote — a test calling StripControlChars
// directly passes even with the wiring removed.
func TestLogOutputHasControlCharsStripped(t *testing.T) {
	got := captureLog(t, func(l *Logger) {
		l.Info("user said %s", "\x00\x1b[31mRED\x07")
	})

	for _, c := range []struct {
		r    rune
		name string
	}{{0x00, "NUL"}, {0x1b, "ESC"}, {0x07, "BEL"}} {
		if strings.ContainsRune(got, c.r) {
			t.Errorf("%s survived into the emitted line; stderr=%q", c.name, got)
		}
	}
	// The ordinary space is legal and survives; only the control bytes go, so the
	// ESC-[ escape is defanged down to the visible text "[31mRED".
	if !strings.Contains(got, "user said [31mRED") {
		t.Errorf("scrubbed line missing expected text; stderr=%q", got)
	}
}

// TestLogOutputKeepsLegalWhitespace pins the other direction: tab/newline/CR are
// LEGAL in a log line, and a scrub that ate them would satisfy "no control chars"
// while mangling every multi-line message.
func TestLogOutputKeepsLegalWhitespace(t *testing.T) {
	const legal = "line1\tcol\nline2\r end"

	got := captureLog(t, func(l *Logger) {
		l.Info("%s", legal)
	})

	if !strings.Contains(got, legal) {
		t.Errorf("legal whitespace did not survive; stderr=%q", got)
	}
}

// captureLog drives a real Logger and returns what it wrote to stderr.
//
// os.Stderr must be swapped BEFORE New(), because the Logger captures the writer
// at construction time — swapping afterwards would leave it pointed at the real
// stderr and the test would read an empty pipe and pass vacuously.
func captureLog(t *testing.T, emit func(*Logger)) string {
	t.Helper()

	Unsuppress()
	SetGlobalLevel(LevelDebug)

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	l := New("inject.test")
	emit(l)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	return buf.String()
}
