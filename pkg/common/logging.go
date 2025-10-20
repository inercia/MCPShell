// Package common provides shared utilities and types used across the MCPShell.
package common

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Global application logger
var globalLogger *Logger

// LogLevel represents logging verbosity levels
type LogLevel int

const (
	// LogLevelNone disables logging
	LogLevelNone LogLevel = iota
	// LogLevelError logs only errors
	LogLevelError
	// LogLevelInfo logs information and errors
	LogLevelInfo
	// LogLevelDebug logs detailed debug information
	LogLevelDebug
)

// LogLevelFromString converts a string representation to a LogLevel
func LogLevelFromString(level string) LogLevel {
	switch level {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "error":
		return LogLevelError
	case "none":
		return LogLevelNone
	default:
		// Default to info level
		return LogLevelInfo
	}
}

// Logger provides a structured logging interface for the application
type Logger struct {
	// The underlying Go logger
	*log.Logger
	// The logging level
	level LogLevel
	// The log file path (if used)
	filePath string
	// The log file handle (if used)
	file *os.File
}

// NewLogger creates a new Logger instance
//
// Parameters:
//   - prefix: The prefix for all log messages
//   - filePath: Path to the log file (empty string disables file logging)
//   - level: The logging verbosity level
//   - truncate: If true, truncate the log file; if false, append to it
//
// Returns:
//   - A new Logger instance
//   - An error if the log file cannot be opened
func NewLogger(prefix string, filePath string, level LogLevel, truncate bool) (*Logger, error) {
	var writer io.Writer
	var file *os.File
	var err error

	// Set up the log writer - always use stderr unless LogLevelNone
	if level == LogLevelNone {
		writer = io.Discard
	} else {
		writer = os.Stderr
	}

	// If a file path is provided, also log to the file
	if filePath != "" && level != LogLevelNone {
		// Determine file open flags
		flags := os.O_RDWR | os.O_CREATE
		if truncate {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_APPEND
		}

		// Open the log file
		file, err = os.OpenFile(filePath, flags, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		// Write to both stderr and file
		writer = io.MultiWriter(os.Stderr, file)
	}

	// Create the logger
	logger := &Logger{
		Logger:   log.New(writer, prefix, log.Ldate|log.Ltime|log.Lshortfile),
		level:    level,
		filePath: filePath,
		file:     file,
	}

	// Log the initialization
	if filePath != "" && level >= LogLevelInfo {
		logger.Debug("----------------------------")
		logger.Debug("Logging initialized to file: %s", filePath)
	}

	return logger, nil
}

// Close closes the log file if it's open
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Debug logs a message at debug level
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level >= LogLevelDebug {
		l.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs a message at info level
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level >= LogLevelInfo {
		l.Printf("[INFO] "+format, v...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level >= LogLevelInfo {
		l.Printf("[WARN] "+format, v...)
	}
}

// Error logs a message at error level
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level >= LogLevelError {
		l.Printf("[ERROR] "+format, v...)
	}
}

// FilePath returns the current log file path
func (l *Logger) FilePath() string {
	return l.filePath
}

// Level returns the current log level
func (l *Logger) Level() LogLevel {
	return l.level
}

// SetLevel changes the current log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

//////////////////////////////////////////////////////////////////////

// GetLogger returns the global application logger.
// If the logger hasn't been initialized yet, it returns a default stderr logger.
func GetLogger() *Logger {
	if globalLogger == nil {
		// Create a default stderr logger at info level
		logger, err := NewLogger("[mcpshell] ", "", LogLevelInfo, false)
		if err != nil {
			// If we can't even create a basic logger, just return a minimal one
			fmt.Fprintf(os.Stderr, "Error creating default logger: %v\n", err)
			minimalLogger, _ := NewLogger("[mcpshell] ", "", LogLevelError, false)
			return minimalLogger
		}
		return logger
	}
	return globalLogger
}

// SetLogger sets the global application logger
func SetLogger(logger *Logger) {
	globalLogger = logger
}
