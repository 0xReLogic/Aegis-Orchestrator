package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// LevelDebug for detailed troubleshooting
	LevelDebug LogLevel = iota
	// LevelInfo for general operational information
	LevelInfo
	// LevelWarn for non-critical issues
	LevelWarn
	// LevelError for errors that should be addressed
	LevelError
	// LevelFatal for critical errors that require immediate attention
	LevelFatal
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// Logger is the interface for logging messages
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
}

// StandardLogger is a simple implementation of the Logger interface
type StandardLogger struct {
	level  LogLevel
	output io.Writer
}

// NewLogger creates a new StandardLogger with the specified log level
func NewLogger(level LogLevel) *StandardLogger {
	return &StandardLogger{
		level:  level,
		output: os.Stdout,
	}
}

// SetLevel sets the minimum log level that will be output
func (l *StandardLogger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput sets the output destination for the logger
func (l *StandardLogger) SetOutput(w io.Writer) {
	l.output = w
}

// log outputs a log message if the level is sufficient
func (l *StandardLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	fmt.Fprintf(l.output, "[%s] [%s] %s\n", timestamp, levelName, message)

	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *StandardLogger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an informational message
func (l *StandardLogger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *StandardLogger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *StandardLogger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a fatal message and exits the program
func (l *StandardLogger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
}

// ParseLogLevel converts a string log level to a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// Global logger instance
var defaultLogger = NewLogger(LevelInfo)

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger *StandardLogger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() *StandardLogger {
	return defaultLogger
}

// Global convenience functions that use the default logger

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs an informational message using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a fatal message and exits the program using the default logger
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}
