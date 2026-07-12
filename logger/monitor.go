package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
)

const DefaultTextLimit = 500

func Request(component string, operation string, attrs ...slog.Attr) {
	Log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"request received",
		append(baseAttrs(component, operation), attrs...)...,
	)
}

func Response(component string, operation string, startedAt time.Time, attrs ...slog.Attr) {
	Log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"response generated",
		append(
			append(baseAttrs(component, operation), slog.Int64("latency_ms", time.Since(startedAt).Milliseconds())),
			attrs...,
		)...,
	)
}

func Failure(component string, operation string, startedAt time.Time, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	Log.LogAttrs(
		context.Background(),
		slog.LevelError,
		"request failed",
		append(
			append(baseAttrs(component, operation), slog.Int64("latency_ms", time.Since(startedAt).Milliseconds())),
			attrs...,
		)...,
	)
}

func InfoPayload(component string, operation string, message string, attrs ...slog.Attr) {
	Log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		message,
		append(baseAttrs(component, operation), attrs...)...,
	)
}

func CompactJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}

	return string(raw)
}

func PrettyJSON(value any) string {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "[]"
	}

	return string(raw)
}

func DebugPayload(component string, operation string, attrs ...slog.Attr) {
	Log.LogAttrs(
		context.Background(),
		slog.LevelDebug,
		"debug payload",
		append(baseAttrs(component, operation), attrs...)...,
	)
}

func Truncate(value string, limit int) string {
	value = strings.Join(
		strings.Fields(
			strings.TrimSpace(value),
		),
		" ",
	)

	if limit <= 0 {
		limit = DefaultTextLimit
	}

	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return string(runes[:limit]) + "..."
}

func baseAttrs(component string, operation string) []slog.Attr {
	return []slog.Attr{
		slog.String("component", component),
		slog.String("operation", operation),
	}
}
