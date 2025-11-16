package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

var defaultLogger *log.Logger

func init() {
	defaultLogger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "2006/01/02 15:04:05",
	})
}

// GetLogger returns the default logger instance
func GetLogger() *log.Logger {
	return defaultLogger
}

// SetLevel sets the log level for the default logger
func SetLevel(level log.Level) {
	defaultLogger.SetLevel(level)
}

// Debug logs a debug message
func Debug(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Debug(msg, keyvals...)
}

// Info logs an info message
func Info(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Info(msg, keyvals...)
}

// Warn logs a warning message
func Warn(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Warn(msg, keyvals...)
}

// Error logs an error message
func Error(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and exits
func Fatal(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Fatal(msg, keyvals...)
}

// Printf logs a message using Printf-style formatting (for compatibility)
func Printf(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Println logs a message (for compatibility)
func Println(msg ...interface{}) {
	if len(msg) == 0 {
		return
	}
	defaultLogger.Info(msg[0])
}

// Fatalf logs a fatal message with Printf-style formatting and exits
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}
