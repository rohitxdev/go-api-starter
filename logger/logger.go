package logger

import (
	"io"
	"log/slog"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

// New creates a new logger. If pretty is true, it will print the logs in a colorful, human-readable format. Pretty should only be used for development.
func New(w io.Writer, isPretty bool) *zerolog.Logger {
	if isPretty {
		prettyWriter := zerolog.NewConsoleWriter(func(cw *zerolog.ConsoleWriter) {
			cw.TimeFormat = time.Kitchen
			cw.Out = w
		})
		z := zerolog.New(prettyWriter).With().Timestamp().Logger()
		return &z
	}
	asyncWriter := diode.NewWriter(w, 10000, 0, func(missed int) {
		slogHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.New(slogHandler).Error("Zerolog: Dropped logs due to slow writer", "dropped", missed)
	})
	z := zerolog.New(asyncWriter).With().Timestamp().Logger()
	return &z
}
