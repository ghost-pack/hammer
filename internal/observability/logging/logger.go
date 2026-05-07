package logging

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

type Format string

const (
	FormatAuto Format = "auto"
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Options struct {
	Level     slog.Level
	Format    Format
	AddSource bool
	Project   string
	Writer    *os.File
}

func New() *slog.Logger {
	return NewWithOptions(optionsFromEnv())
}

func NewWithOptions(opts Options) *slog.Logger {
	if opts.Writer == nil {
		opts.Writer = os.Stderr
	}

	format := opts.Format
	if format == "" || format == FormatAuto {
		format = autoDetectFormat(opts.Writer)
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: replaceAttrFor(format, opts.Project),
	}

	var handler slog.Handler
	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(opts.Writer, handlerOpts)
	case FormatText:
		handler = tint.NewHandler(opts.Writer, &tint.Options{
			Level:      opts.Level,
			AddSource:  opts.AddSource,
			TimeFormat: time.Kitchen,
			NoColor:    !isatty.IsTerminal(os.Stdout.Fd()),
		})
	default:
		handler = slog.NewJSONHandler(opts.Writer, handlerOpts)
	}

	handler = &traceHandler{
		Handler: handler,
		project: opts.Project,
	}
	return slog.New(handler)
}

func optionsFromEnv() Options {
	level := parseLevel(os.Getenv("LOG_LEVEL"), slog.LevelInfo)
	format := Format(strings.ToLower(os.Getenv("LOG_FORMAT")))
	if format == "" {
		format = FormatAuto
	}
	addSource := level == slog.LevelDebug
	if v := os.Getenv("LOG_SOURCE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			addSource = b
		}
	}

	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	return Options{
		Level:     level,
		Format:    format,
		AddSource: addSource,
		Project:   project,
	}
}

func parseLevel(s string, fallback slog.Level) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}

func autoDetectFormat(w *os.File) Format {
	if w == nil {
		return FormatJSON
	}
	fi, err := w.Stat()
	if err != nil {
		return FormatJSON
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		return FormatText
	}
	return FormatJSON
}
