package logger

import (
	"io"
	"log/slog"
	"os"
)

var Logger *Logger

func init() {
	Logger = New(os.Getenv("DEBUG") == "1")
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
