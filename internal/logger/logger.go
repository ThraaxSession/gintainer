package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	logBuffer   *RingBuffer
)

// Log level constants
const (
	DebugLevel = log.DebugLevel
	InfoLevel  = log.InfoLevel
	WarnLevel  = log.WarnLevel
	ErrorLevel = log.ErrorLevel
	FatalLevel = log.FatalLevel
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// RingBuffer holds recent log entries
type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	maxSize int
	pos     int
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, 0, size),
		maxSize: size,
	}
}

// Add adds a log entry to the buffer
func (rb *RingBuffer) Add(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.entries) < rb.maxSize {
		rb.entries = append(rb.entries, entry)
	} else {
		rb.entries[rb.pos] = entry
		rb.pos = (rb.pos + 1) % rb.maxSize
	}
}

// GetAll returns all log entries
func (rb *RingBuffer) GetAll() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]LogEntry, len(rb.entries))
	if len(rb.entries) < rb.maxSize {
		copy(result, rb.entries)
	} else {
		// Copy from pos to end
		n := copy(result, rb.entries[rb.pos:])
		// Copy from start to pos
		copy(result[n:], rb.entries[:rb.pos])
	}
	return result
}

// TeeWriter wraps an io.Writer and captures log output
type TeeWriter struct {
	writer io.Writer
	buffer *RingBuffer
	level  string
}

func (t *TeeWriter) Write(p []byte) (n int, err error) {
	// Write to original writer
	n, err = t.writer.Write(p)

	// Also capture in buffer
	if t.buffer != nil {
		msg := string(bytes.TrimSpace(p))
		if msg != "" {
			t.buffer.Add(LogEntry{
				Timestamp: time.Now(),
				Level:     t.level,
				Message:   msg,
			})
		}
	}

	return n, err
}

// GetLogBuffer returns the log buffer
func GetLogBuffer() *RingBuffer {
	return logBuffer
}

func init() {
	// Initialize log buffer (keep last 1000 log entries)
	logBuffer = NewRingBuffer(1000)

	// Create tee writers to capture logs
	stdoutTee := &TeeWriter{writer: os.Stdout, buffer: logBuffer, level: "INFO"}
	stderrTee := &TeeWriter{writer: os.Stderr, buffer: logBuffer, level: "ERROR"}

	// Info and debug messages go to stdout
	infoLogger = log.NewWithOptions(stdoutTee, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "2006/01/02 15:04:05",
	})

	// Warn, error, and fatal messages go to stderr
	errorLogger = log.NewWithOptions(stderrTee, log.Options{
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

// FormatLogEntry formats a log entry for display
func FormatLogEntry(entry LogEntry) string {
	return fmt.Sprintf("%s [%s] %s",
		entry.Timestamp.Format("2006/01/02 15:04:05"),
		entry.Level,
		entry.Message)
}
