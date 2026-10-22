package identity

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/teatchable/backend/internal/store"
)

// staleSessionRetention keeps expired/revoked sessions around briefly for audits.
const staleSessionRetention = 7 * 24 * time.Hour

type CleanupSessionsArgs struct{}

func (CleanupSessionsArgs) Kind() string { return "identity.cleanup_sessions" }

type CleanupSessionsWorker struct {
	river.WorkerDefaults[CleanupSessionsArgs]
	Pool *pgxpool.Pool
}

func (w *CleanupSessionsWorker) Work(ctx context.Context, _ *river.Job[CleanupSessionsArgs]) error {
	n, err := store.New(w.Pool).DeleteStaleSessions(ctx, time.Now().Add(-staleSessionRetention))
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "deleted stale sessions", "count", n)
	return nil
}
