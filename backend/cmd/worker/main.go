// Command worker runs background jobs (River) and periodic tasks.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &identity.CleanupSessionsWorker{Pool: pool})

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:  log,
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 20}},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) { return identity.CleanupSessionsArgs{}, nil },
				&river.PeriodicJobOpts{RunOnStart: true}),
		},
	})
	if err != nil {
		return err
	}

	if err := client.Start(ctx); err != nil {
		return err
	}
	log.Info("worker started")
	<-ctx.Done()

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return client.Stop(shutdownCtx)
}
