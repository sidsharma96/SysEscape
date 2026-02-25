// Package log provides structured JSON logging via slog.
package log

import (
	"log/slog"
	"os"
)

// NewLogger returns a new slog.Logger configured for structured JSON output.
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
