package logger

import (
	"io"
	"log/slog"
	"os"
)

var DefaultLogger *Logger

func init() {
	DefaultLogger = New(os.Getenv("DEBUG") == "1")
}

// Info logs an informational message
func Info(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Info(msg, keysAndValues...)
}

// Error logs an error message
func Error(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Error(msg, keysAndValues...)
}

// Logger wraps slog.Logger to provide structured logging
type Logger struct {
	*slog.Logger
}

// New creates a new Logger instance
func New(debug bool) *Logger {
	var handler slog.Handler
	if debug {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	
	return &Logger{
		Logger: slog.New(handler),
	}
}
