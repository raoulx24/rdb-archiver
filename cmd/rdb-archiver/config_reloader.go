package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
)

type ConfigReloader struct {
	file     string
	method   string
	fw       *watchfs.FileWatcher
	logg     logging.Logger
	apply    func(newCfg *config.Config)
	cancel   context.CancelFunc
	timer    *time.Timer
	reloadCh chan struct{}
}

func NewConfigReloader(
	file string,
	method string,
	fw *watchfs.FileWatcher,
	logg logging.Logger,
	apply func(newCfg *config.Config),
) *ConfigReloader {
	return &ConfigReloader{
		file:     file,
		method:   method,
		fw:       fw,
		logg:     logg,
		apply:    apply,
		reloadCh: make(chan struct{}, 1),
	}
}

func (r *ConfigReloader) Start(ctx context.Context) {
	r.startWatcher(ctx)

	for {
		select {
		case <-ctx.Done():
			if r.cancel != nil {
				r.cancel()
			}
			return

		case <-r.reloadCh:
			r.scheduleReload(ctx)
		}
	}
}

func (r *ConfigReloader) startWatcher(ctx context.Context) {
	if r.cancel != nil {
		r.cancel()
	}

	var wctx context.Context
	wctx, r.cancel = context.WithCancel(ctx)

	dir := filepath.Dir(r.file)
	base := filepath.Base(r.file)

	go func() {
		if err := r.fw.StartWatchingForFile(wctx, r.method, dir, base, r.reloadCh); err != nil {
			r.logg.Error("config watcher failed", "error", err)
		}
	}()
}

func (r *ConfigReloader) scheduleReload(ctx context.Context) {
	if r.timer != nil {
		r.timer.Stop()
	}

	r.timer = time.AfterFunc(300*time.Millisecond, func() {
		newCfg, err := config.Load(r.file)
		if err != nil {
			r.logg.Error("config reload failed", "error", err)
			return
		}
		newCfg.ApplyDefaults()

		r.apply(newCfg)

		if newCfg.ConfigReload.Method != r.method {
			r.method = newCfg.ConfigReload.Method
			r.startWatcher(ctx)
		}

		r.logg.Info("config reloaded")
	})
}
