package utils

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger is the interface for logging
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger
}

// logrusLogger wraps logrus.Entry to implement our Logger interface
type logrusLogger struct {
	entry *logrus.Entry
}

// NewLogger creates a new logger instance
func NewLogger(serviceName string, logLevel string, format string) Logger {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set formatter
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	log.SetOutput(os.Stdout)

	return &logrusLogger{
		entry: log.WithField("service", serviceName),
	}
}

// NewTestLogger creates a logger for testing that discards output
func NewTestLogger() Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return &logrusLogger{
		entry: logrus.NewEntry(log),
	}
}

// Debug logs a debug message
func (l *logrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info logs an info message
func (l *logrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Warn logs a warning message
func (l *logrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error logs an error message
func (l *logrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Fatal logs a fatal message and exits
func (l *logrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Debugf logs a formatted debug message
func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *logrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// WithField adds a single field to the logger
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		entry: l.entry.WithField(key, value),
	}
}

// WithFields adds multiple fields to the logger
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &logrusLogger{
		entry: l.entry.WithFields(logrus.Fields(fields)),
	}
}

// WithError adds an error field to the logger
func (l *logrusLogger) WithError(err error) Logger {
	return &logrusLogger{
		entry: l.entry.WithError(err),
	}
}

// WithContext adds context fields to the logger
func (l *logrusLogger) WithContext(ctx context.Context) Logger {
	entry := l.entry

	// Extract trace ID from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}

	// Extract request ID from context if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	return &logrusLogger{entry: entry}
}

// Global logger instance
var globalLogger Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(serviceName string, logLevel string, format string) {
	globalLogger = NewLogger(serviceName, logLevel, format)
}

// GetLogger returns the global logger
func GetLogger() Logger {
	if globalLogger == nil {
		globalLogger = NewLogger("default", "info", "text")
	}
	return globalLogger
}

