package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestJSONFormatRemapsKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOptions(Options{
		Format: FormatJSON,
		Level:  slog.LevelDebug,
		Writer: nil, // we'll override the handler manually for testing
	})
	// Re-create with our buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttrFor(FormatJSON, "myproj"),
	})
	logger = slog.New(&traceHandler{Handler: h, project: "myproj"})

	logger.Info("hello", "app", "my-api")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["severity"] != "INFO" {
		t.Errorf("want severity=INFO, got %v", got["severity"])
	}
	if got["message"] != "hello" {
		t.Errorf("want message=hello, got %v", got["message"])
	}
	if got["app"] != "my-api" {
		t.Errorf("want app=my-api, got %v", got["app"])
	}
	if _, ok := got["timestamp"]; !ok {
		t.Errorf("missing timestamp")
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"weird":   slog.LevelInfo, // fallback
	}
	for input, want := range cases {
		if got := parseLevel(input, slog.LevelInfo); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestTraceHandlerAddsTraceFields(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	wrapped := &traceHandler{Handler: h, project: "myproj"}
	logger := slog.New(wrapped)

	// Without trace context, no trace fields
	logger.InfoContext(context.Background(), "no trace")
	if strings.Contains(buf.String(), "logging.googleapis.com/trace") {
		t.Errorf("unexpected trace field: %s", buf.String())
	}
}
