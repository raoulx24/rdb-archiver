package main

import (
	"context"
	"net/http"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/metrics/prometheus"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

type MetricsBundle struct {
	Worker          worker.Metrics
	SnapshotWatcher snapshotwatcher.Metrics
	Mailbox         mailbox.Metrics
	Retention       retention.Metrics
}

func startPrometheus(ctx context.Context, cfg prometheus.Config, logg logging.Logger, mux http.Handler) (prometheus.Server, context.CancelFunc) {
	srv := prometheus.New(cfg, logg, mux)
	srvCtx, cancel := context.WithCancel(ctx)

	go func() {
		if err := srv.Start(srvCtx); err != nil {
			logg.Error("metrics server stopped", "error", err)
		}
	}()

	return srv, cancel
}

func buildMetrics(cfg prometheus.Config) (*http.ServeMux, MetricsBundle) {
	reg := prometheus.NewRegistry()

	bundle := MetricsBundle{
		Worker:          prometheus.NewWorkerMetrics(reg, cfg),
		SnapshotWatcher: prometheus.NewSnapshotWatcherMetrics(reg, cfg),
		Mailbox:         prometheus.NewMailboxMetrics(reg),
		Retention:       prometheus.NewRetentionMetrics(reg, cfg),
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.Handler())

	return mux, bundle
}
