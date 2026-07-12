package logger

import (
	"log/slog"
	"os"
	"strings"
)

var Log = slog.New(
	slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: logLevel(),
		},
	),
)

func logLevel() slog.Level {
	switch strings.ToLower(
		strings.TrimSpace(os.Getenv("LOG_LEVEL")),
	) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
