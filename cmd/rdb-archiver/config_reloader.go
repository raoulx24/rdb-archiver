package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/watchfs"
)

type ConfigReloader struct {
	file     string
	fileHash string
	method   string
	fw       *watchfs.FileWatcher
	logg     logging.Logger
	apply    func(newCfg *config.Config)
	cancel   context.CancelFunc
	timer    *time.Timer
	mu       sync.Mutex
	reloadCh chan struct{}
}

func NewConfigReloader(
	file string,
	fileHash string,
	method string,
	fw *watchfs.FileWatcher,
	logg logging.Logger,
	apply func(newCfg *config.Config),
) *ConfigReloader {
	return &ConfigReloader{
		file:     file,
		fileHash: fileHash,
		method:   method,
		fw:       fw,
		logg:     logg,
		apply:    apply,
		reloadCh: make(chan struct{}, 1),
	}
}

func (r *ConfigReloader) Start(ctx context.Context) {
	r.startWatcher(ctx)
	// check that the config file did not change between calls
	// it forces a synthetic reload. if hash is the same, nothing happens
	r.scheduleReload(ctx)

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
	// normally, we should wait for the old watcher to stop. this should be done using a
	// `done := make(chan struct{})` in fsnotify and polling, adding a `defer close(done)`
	// in go routine loops and returning it at the end of the functions. this is not (yet) done
	// as there are checks here, so if 2 watchers are running in the same time, it would be ok,
	// as only one can trigger the reload

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
		r.mu.Lock()
		newCfg, fileHash, err := config.Load(r.file)
		if err != nil {
			r.logg.Error("config reload failed", "error", err)
			r.mu.Unlock()
			return
		}
		if r.fileHash == fileHash {
			// file did not change
			r.mu.Unlock()
			return
		}
		r.fileHash = fileHash
		r.mu.Unlock()

		r.logg.Info("config file change detected")

		stdLog := log.New(os.Stdout, "", log.LstdFlags)
		newCfg.ApplyDefaults(stdLog)

		r.apply(newCfg)

		if newCfg.ConfigReload.Method != r.method {
			r.method = newCfg.ConfigReload.Method
			r.startWatcher(ctx)
		}
	})
}
