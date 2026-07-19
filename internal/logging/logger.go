package logging

import (
	"io"
	"log/slog"
	"strings"
)

func New(output io.Writer, level string) *slog.Logger {
	selected := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		selected = slog.LevelDebug
	case "warn", "warning":
		selected = slog.LevelWarn
	case "error":
		selected = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: selected}))
}
