package logger

import (
	"log/slog"
	"os"
)

// New cria um logger JSON estruturado, com nível configurável.
// Logs estruturados facilitam correlacionar por tower_id / request_id,
// tal como já é prática no towercore (ver runbook.md).
func New(level string) *slog.Logger {
	var lvl slog.Level

	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(handler)
}
