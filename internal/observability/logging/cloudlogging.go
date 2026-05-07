package logging

import (
	"log/slog"
	"time"
)

func replaceAttrFor(format Format, project string) func(groups []string, a slog.Attr) slog.Attr {
	if format != FormatJSON {
		return nil
	}

	return func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) > 0 {
			return a
		}
		switch a.Key {
		case slog.LevelKey:
			lvl := a.Value.Any().(slog.Level)
			return slog.String("severity", levelToSeverity(lvl))
		case slog.MessageKey:
			return slog.String("message", a.Value.String())
		case slog.TimeKey:
			t := a.Value.Time()
			return slog.String("timestamp", t.Format(time.RFC3339))
		case slog.SourceKey:
			return a
		}
		return a
	}
}

func levelToSeverity(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
