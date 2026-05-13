package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type Options struct {
	Dir string
	Env []string
}

type Runner interface {
	Run(ctx context.Context, name string, args []string, opts Options) (*Result, error)
	RunWithoutOptions(ctx context.Context, name string, args []string) (*Result, error)
}

type OSRunner struct{}

func New() *OSRunner {
	return &OSRunner{}
}

func (*OSRunner) Run(ctx context.Context, name string, args []string, opts Options) (*Result, error) {
	ctx, span := tracing.Tracer("runner").Start(ctx, "exec:"+name,
		trace.WithAttributes(
			attribute.String("cmd", name),
			attribute.String("dir", opts.Dir),
			attribute.StringSlice("args", args)))
	defer span.End()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	return runCommand(cmd, &outBuf, &errBuf, span, name)
}

func (*OSRunner) RunWithoutOptions(ctx context.Context, name string, args []string) (*Result, error) {
	ctx, span := tracing.Tracer("runner").Start(ctx, "exec:"+name,
		trace.WithAttributes(
			attribute.String("cmd", name),
			attribute.StringSlice("args", args)))
	defer span.End()

	cmd := exec.CommandContext(ctx, name, args...)

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	return runCommand(cmd, &outBuf, &errBuf, span, name)
}

func runCommand(cmd *exec.Cmd, outBuf *bytes.Buffer, errBuf *bytes.Buffer, span trace.Span, name string) (*Result, error) {
	runErr := cmd.Run()
	res := &Result{
		Stdout:   outBuf.Bytes(),
		Stderr:   errBuf.Bytes(),
		ExitCode: cmd.ProcessState.ExitCode(),
	}

	span.SetAttributes(attribute.Int("exit_code", res.ExitCode))

	if _, ok := errors.AsType[*exec.ExitError](runErr); ok {
		span.SetStatus(codes.Error, "non-zero exit")
		return res, fmt.Errorf("%s exited with code %d", name, res.ExitCode)
	}
	if runErr != nil {
		span.RecordError(runErr)
		span.SetStatus(codes.Error, runErr.Error())
		return res, fmt.Errorf("failed to run %s: %w", name, runErr)
	}
	return res, nil
}
