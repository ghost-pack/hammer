package tracing

import (
	"go.opentelemetry.io/otel/attribute"
)

// Common attribute keys used across hammer spans.
// Keep these in sync with logging.* attribute helpers.
const (
	AttrApp        = "hammer.app"
	AttrKind       = "hammer.kind"
	AttrEnv        = "hammer.env"
	AttrTeam       = "hammer.team"
	AttrCommit     = "hammer.commit"
	AttrPhase      = "hammer.phase"
	AttrCommand    = "hammer.command"
	AttrToolName   = "hammer.tool.name" // e.g., "trivy", "ko"
	AttrToolArgs   = "hammer.tool.args"
	AttrExitCode   = "hammer.tool.exit_code"
	AttrDurationMs = "hammer.duration_ms"
)

func App(name string) attribute.KeyValue        { return attribute.String(AttrApp, name) }
func Kind(kind string) attribute.KeyValue       { return attribute.String(AttrKind, kind) }
func Env(env string) attribute.KeyValue         { return attribute.String(AttrEnv, env) }
func Team(team string) attribute.KeyValue       { return attribute.String(AttrTeam, team) }
func Commit(sha string) attribute.KeyValue      { return attribute.String(AttrCommit, sha) }
func Phase(phase string) attribute.KeyValue     { return attribute.String(AttrPhase, phase) }
func Command(cmd string) attribute.KeyValue     { return attribute.String(AttrCommand, cmd) }
func ToolName(name string) attribute.KeyValue   { return attribute.String(AttrToolName, name) }
func ToolArgs(args []string) attribute.KeyValue { return attribute.StringSlice(AttrToolArgs, args) }
func ExitCode(code int) attribute.KeyValue      { return attribute.Int(AttrExitCode, code) }
