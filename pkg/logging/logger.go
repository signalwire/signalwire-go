// Package logging provides structured logging for the SignalWire AI Agents SDK.
//
// It supports log levels (debug, info, warn, error), named loggers per component,
// and can be suppressed globally for CLI tools or testing.
package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents a log severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelOff:   "OFF",
}

// ParseLevel converts a string level name to a Level.
// Returns LevelInfo if the string is not recognized.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "off", "none", "silent":
		return LevelOff
	default:
		return LevelInfo
	}
}

var (
	globalLevel Level = LevelInfo
	globalMu    sync.RWMutex
	suppressed  bool
)

// configureFromEnv reads SIGNALWIRE_LOG_LEVEL and SIGNALWIRE_LOG_MODE from the
// environment and applies them to globalLevel and suppressed. Must be called
// with globalMu held for writing, or before any goroutines are started (i.e.
// from init or ResetLoggingConfiguration).
func configureFromEnv() {
	globalLevel = LevelInfo
	suppressed = false
	if envLevel := os.Getenv("SIGNALWIRE_LOG_LEVEL"); envLevel != "" {
		globalLevel = ParseLevel(envLevel)
	}
	if envMode := os.Getenv("SIGNALWIRE_LOG_MODE"); strings.ToLower(envMode) == "off" {
		suppressed = true
	}
}

func init() {
	configureFromEnv()
}

// ResetLoggingConfiguration re-reads SIGNALWIRE_LOG_LEVEL and SIGNALWIRE_LOG_MODE
// from the environment and resets globalLevel and suppressed to the env-derived
// defaults. It is intended for test teardown and env-var-driven
// reconfiguration at runtime.
func ResetLoggingConfiguration() {
	globalMu.Lock()
	defer globalMu.Unlock()
	configureFromEnv()
}

// SetGlobalLevel sets the minimum log level for all loggers.
func SetGlobalLevel(level Level) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLevel = level
}

// GetGlobalLevel returns the current global log level.
func GetGlobalLevel() Level {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLevel
}

// Suppress disables all log output.
func Suppress() {
	globalMu.Lock()
	defer globalMu.Unlock()
	suppressed = true
}

// Unsuppress re-enables log output.
func Unsuppress() {
	globalMu.Lock()
	defer globalMu.Unlock()
	suppressed = false
}

// IsSuppressed returns whether logging is currently suppressed.
func IsSuppressed() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return suppressed
}

// isControlChar reports whether r is one of the control characters the
// reference strips: the C0 range minus tab/newline/carriage-return, plus DEL and
// the C1 range. Mirrors the reference's `_CONTROL_CHAR_RE`
// (`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f-\x9f]`) — tab (0x09), LF (0x0a) and CR (0x0d)
// are legal in a log line and deliberately survive.
func isControlChar(r rune) bool {
	return (r >= 0x00 && r <= 0x08) ||
		r == 0x0b || r == 0x0c ||
		(r >= 0x0e && r <= 0x1f) ||
		(r >= 0x7f && r <= 0x9f)
}

// stripControlCharsValue removes control characters from a single string. This is
// the unit the log emitter needs: StripControlChars is the reference's event-map
// contract, but a line-oriented writer has one string, not a map. Both go through
// this function so the two can never diverge.
func stripControlCharsValue(value string) string {
	return strings.Map(func(r rune) rune {
		if isControlChar(r) {
			return -1
		}
		return r
	}, value)
}

// StripControlChars removes control characters from every string value of a log
// event map, preventing log injection. Non-string values pass through untouched
// (the reference's `isinstance(value, str)` guard). The map is modified in place
// and returned, mirroring the reference's structlog processor.
func StripControlChars(eventDict map[string]any) map[string]any {
	for key, value := range eventDict {
		if s, ok := value.(string); ok {
			eventDict[key] = stripControlCharsValue(s)
		}
	}
	return eventDict
}

// Logger is a named logger that respects global log level settings.
type Logger struct {
	name   string
	logger *log.Logger
}

// New creates a new Logger with the given component name.
func New(name string) *Logger {
	return &Logger{
		name:   name,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (l *Logger) shouldLog(level Level) bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return !suppressed && level >= globalLevel
}

func (l *Logger) log(level Level, format string, args ...any) {
	if !l.shouldLog(level) {
		return
	}
	prefix := fmt.Sprintf("[%s] [%s] ", levelNames[level], l.name)
	// Scrub control characters BEFORE emitting — log-injection defence, and the
	// reason the reference registers strip_control_chars in both of its structlog
	// processor chains. Scrubbing the FORMATTED message (not the format string)
	// is what matters: the injected bytes arrive through args at every real call
	// site, so scrubbing earlier would miss them entirely.
	msg := stripControlCharsValue(fmt.Sprintf(format, args...))
	l.logger.Print(prefix + msg)
}

// Debug logs a message at debug level.
func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

// Info logs a message at info level.
func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a message at warn level.
func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarn, format, args...)
}

// Error logs a message at error level.
func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, format, args...)
}
