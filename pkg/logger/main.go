package logger

import (
	"log/slog"
)

var Record *slog.Logger

func init() {
	Record = slog.Default().WithGroup("cardano-valley")
	Record.Info("LOGGER", "INITIALIZED", true)
}