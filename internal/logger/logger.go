package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

// New creates a new logger. If pretty is true, it will print the logs in a colorful, human-readable format. Pretty should only be used for development.
func New(w io.Writer, pretty bool) *zerolog.Logger {
	if pretty {
		prettyWriter := zerolog.NewConsoleWriter(func(cw *zerolog.ConsoleWriter) {
			cw.TimeFormat = time.Kitchen
			cw.Out = w
		})
		z := zerolog.New(prettyWriter).With().Timestamp().Logger()
		return &z
	}
	asyncWriter := diode.NewWriter(w, 10000, 0, func(missed int) {
		slog.Error("Zerolog: Dropped logs due to slow writer", "dropped", missed)
		os.Exit(1)
	})
	z := zerolog.New(asyncWriter).With().Timestamp().Logger()
	return &z
}
