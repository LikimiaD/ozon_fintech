package logger

import (
	"log/slog"
	"os"
)

// FatalError logs a fatal error message and terminates the program.
func FatalError(text string, err error) {
	slog.Error(text, "error", err.Error())
	os.Exit(1)
}
