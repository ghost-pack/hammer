package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	Dir        string
	Env        []string
	StreamTo   io.Writer
	InheritEnv bool
}

func Run(ctx context.Context, name string, args []string, opts Options) (*Result, error) {
	ctx, span := tracing.Tracer("runner").Start(ctx, "exec:"+name,
		trace.WithAttributes(
			attribute.String("cmd", name),
			attribute.String("dir", opts.Dir),
			attribute.StringSlice("args", args)))
	defer span.End()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	if opts.InheritEnv {
		cmd.Env = append(os.Environ(), opts.Env...)
	} else {
		cmd.Env = opts.Env
	}

	var outBuf, errBuf bytes.Buffer
	if opts.StreamTo != nil {
		cmd.Stdout = io.MultiWriter(opts.StreamTo, &outBuf)
		cmd.Stderr = io.MultiWriter(opts.StreamTo, &errBuf)
	} else {
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
	}

	runErr := cmd.Run()
	res := &Result{
		Stdout:   outBuf.Bytes(),
		Stderr:   errBuf.Bytes(),
		ExitCode: cmd.ProcessState.ExitCode(),
	}

	span.SetAttributes(attribute.Int("exit_code", res.ExitCode))

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		span.SetStatus(codes.Error, "non-zero exit")
		return res, fmt.Errorf("%s exited with code %d", name, res.ExitCode)
	}
	if runErr != nil {
		span.RecordError(runErr)
		span.SetStatus(codes.Error, runErr.Error())
		return res, fmt.Errorf("Failed to run %s: %w", name, runErr)
	}
	return res, nil
}
