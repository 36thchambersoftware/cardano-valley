package logger

import (
	"log/slog"
	"os"
)

var Record *slog.Logger

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	Record = logger.WithGroup("CV")
	Record.Info("LOGGER", "INITIALIZED", true)
}