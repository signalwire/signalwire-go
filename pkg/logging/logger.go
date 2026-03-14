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

func init() {
	// Check environment for log level and mode
	if envLevel := os.Getenv("SIGNALWIRE_LOG_LEVEL"); envLevel != "" {
		globalLevel = ParseLevel(envLevel)
	}
	if envMode := os.Getenv("SIGNALWIRE_LOG_MODE"); strings.ToLower(envMode) == "off" {
		suppressed = true
	}
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
	msg := fmt.Sprintf(format, args...)
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
