package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/adamantal/prmoji/internal/store"
)

func CutoffDateUTC(now time.Time, days int) time.Time {
	today := now.UTC().Truncate(24 * time.Hour)
	return today.AddDate(0, 0, -days)
}

func Run(ctx context.Context, st *store.SQLiteStore, retentionDays int, now time.Time) (int64, error) {
	slog.Info("running cleanup", "retention_days", retentionDays)
	return st.DeleteOlderThanDate(ctx, CutoffDateUTC(now, retentionDays))
}
