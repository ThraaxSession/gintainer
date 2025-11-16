package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

func init() {
	// Info and debug messages go to stdout
	infoLogger = log.NewWithOptions(os.Stdout, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "2006/01/02 15:04:05",
	})

	// Warn, error, and fatal messages go to stderr
	errorLogger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "2006/01/02 15:04:05",
	})
}

// GetLogger returns the info logger instance (for stdout)
func GetLogger() *log.Logger {
	return infoLogger
}

// SetLevel sets the log level for both loggers
func SetLevel(level log.Level) {
	infoLogger.SetLevel(level)
	errorLogger.SetLevel(level)
}

// Debug logs a debug message to stdout
func Debug(msg interface{}, keyvals ...interface{}) {
	infoLogger.Debug(msg, keyvals...)
}

// Info logs an info message to stdout
func Info(msg interface{}, keyvals ...interface{}) {
	infoLogger.Info(msg, keyvals...)
}

// Warn logs a warning message to stderr
func Warn(msg interface{}, keyvals ...interface{}) {
	errorLogger.Warn(msg, keyvals...)
}

// Error logs an error message to stderr
func Error(msg interface{}, keyvals ...interface{}) {
	errorLogger.Error(msg, keyvals...)
}

// Fatal logs a fatal message to stderr and exits
func Fatal(msg interface{}, keyvals ...interface{}) {
	errorLogger.Fatal(msg, keyvals...)
}

// Printf logs a message using Printf-style formatting (for compatibility) to stdout
func Printf(format string, args ...interface{}) {
	infoLogger.Infof(format, args...)
}

// Println logs a message (for compatibility) to stdout
func Println(msg ...interface{}) {
	if len(msg) == 0 {
		return
	}
	infoLogger.Info(msg[0])
}

// Fatalf logs a fatal message with Printf-style formatting to stderr and exits
func Fatalf(format string, args ...interface{}) {
	errorLogger.Fatalf(format, args...)
}
