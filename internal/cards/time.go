package cards

import (
	"log/slog"
	"time"
)

func TimeTracker(start time.Time, name string) {
	elapsed := time.Since(start)
	slog.Info("time tracker", slog.String("name", name), slog.Duration("time", elapsed))
}
