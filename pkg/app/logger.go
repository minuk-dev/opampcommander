package app

import "log/slog"

func NewLogger() *slog.Logger {
	logger := slog.Default()
	
return logger
}
