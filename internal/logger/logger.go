package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	ansiReset           = "0"
	ansiBold            = "1"
	ansiItalic          = "3"
	ansiUnderline       = "4"
	ansiBlinkSlow       = "5"
	ansiBlinkFast       = "6"
	ansiReverse         = "7"
	ansiHidden          = "8"
	ansiStrikethrough   = "9"
	ansiBlack           = "30"
	ansiRed             = "31"
	ansiGreen           = "32"
	ansiYellow          = "33"
	ansiBlue            = "34"
	ansiMagenta         = "35"
	ansiCyan            = "36"
	ansiWhite           = "37"
	ansiBgBlack         = "40"
	ansiBgRed           = "41"
	ansiBgGreen         = "42"
	ansiBgYellow        = "43"
	ansiBgBlue          = "44"
	ansiBgMagenta       = "45"
	ansiBgCyan          = "46"
	ansiBgWhite         = "47"
	ansiLowIntensity    = "02"
	ansiHighIntensity   = "06"
	ansiBrightBlack     = "90"
	ansiBrightRed       = "91"
	ansiBrightGreen     = "92"
	ansiBrightYellow    = "93"
	ansiBrightBlue      = "94"
	ansiBrightMagenta   = "95"
	ansiBrightCyan      = "96"
	ansiBrightWhite     = "97"
	ansiBgBrightBlack   = "100"
	ansiBgBrightRed     = "101"
	ansiBgBrightGreen   = "102"
	ansiBgBrightYellow  = "103"
	ansiBgBrightBlue    = "104"
	ansiBgBrightMagenta = "105"
	ansiBgBrightCyan    = "106"
	ansiBgBrightWhite   = "107"
	ansiGrey            = "38;5;240"
)

func colorize(ansiCode string, text string) string {
	return "\x1b[" + ansiCode + "m" + text + "\x1b[" + ansiReset + "m"
}

func getLevelColor(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return ansiGrey
	case slog.LevelInfo:
		return ansiBrightCyan
	case slog.LevelWarn:
		return ansiBrightYellow
	case slog.LevelError:
		return ansiBrightRed
	default:
		return ansiBrightWhite
	}
}

func getLevelLabel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DBG"
	case slog.LevelInfo:
		return "INF"
	case slog.LevelWarn:
		return "WRN"
	case slog.LevelError:
		return "ERR"
	default:
		return "???"
	}
}

type prettyLogHandler struct {
	handler slog.Handler
	buf     *bytes.Buffer
	mu      *sync.Mutex
	w       io.Writer
}

var _ slog.Handler = (*prettyLogHandler)(nil)

func (h *prettyLogHandler) computeAttrs(ctx context.Context, r slog.Record) (map[string]any, error) {
	h.mu.Lock()
	defer func() {
		h.buf.Reset()
		h.mu.Unlock()
	}()

	if err := h.handler.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("could not handle in log handler: %w", err)
	}

	var attrs = make(map[string]any)

	if err := json.Unmarshal(h.buf.Bytes(), &attrs); err != nil {
		return nil, fmt.Errorf("could not unmarshal in log handler: %w", err)
	}
	return attrs, nil
}

func (h *prettyLogHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	timeStr := colorize(ansiGrey, r.Time.Format("03:04PM"))
	levelStr := colorize(getLevelColor(r.Level), getLevelLabel(r.Level))
	msgStr := colorize(ansiWhite, r.Message)

	logStr := fmt.Sprintf("%s %s %s", timeStr, levelStr, msgStr)

	//Delete level, msg and time from attributes as they are already printed
	delete(attrs, "level")
	delete(attrs, "msg")
	delete(attrs, "time")

	//If still any attributes, print them
	if len(attrs) > 0 {
		bytes, err := json.MarshalIndent(attrs, "", "  ")
		if err != nil {
			return fmt.Errorf("could not marshal in log handler: %w", err)
		}
		logStr += fmt.Sprintf(" %s\n", colorize(ansiGrey, string(bytes)))
	}

	fmt.Fprintln(h.w, logStr)

	return nil
}

func (h *prettyLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *prettyLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prettyLogHandler{
		handler: h.handler.WithAttrs(attrs),
		buf:     h.buf,
		mu:      h.mu,
	}
}

func (h *prettyLogHandler) WithGroup(name string) slog.Handler {
	return &prettyLogHandler{
		handler: h.handler.WithGroup(name),
		buf:     h.buf,
		mu:      h.mu,
	}
}

type HandlerOptions struct {
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	Level       slog.Level
	NoColor     bool
}

func New(w io.Writer, opts *HandlerOptions) *slog.Logger {
	slogOpts := slog.HandlerOptions{
		Level:       opts.Level,
		ReplaceAttr: opts.ReplaceAttr,
	}
	var handler slog.Handler = slog.NewJSONHandler(w, &slogOpts)
	if !opts.NoColor {
		buf := new(bytes.Buffer)
		handler = &prettyLogHandler{
			//pass buffer instead of given writer to inner handler
			handler: slog.NewJSONHandler(buf, &slogOpts),
			buf:     buf,
			mu:      &sync.Mutex{},
			w:       w,
		}
	}
	return slog.New(handler)
}
