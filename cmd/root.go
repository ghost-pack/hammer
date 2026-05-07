package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghost-pack/hammer/internal/cli"
	"github.com/ghost-pack/hammer/internal/observability/logging"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
)

const name = "github.com/ghost-pack/hammer"

func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := logging.New()
	slog.SetDefault(logger)

	shutdown, err := tracing.Init(ctx)
	if err != nil {
		slog.WarnContext(ctx, "tracing init failed", "err", err)
	}
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(sctx); err != nil {
			slog.WarnContext(ctx, "tracing shutdown failed", "err", err)
		}
	}()

	if err := cli.Execute(ctx); err != nil {
		slog.ErrorContext(ctx, "command failed", "err", err)
		return err
	}
	return nil
}
