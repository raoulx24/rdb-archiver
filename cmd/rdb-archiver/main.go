package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/raoulx24/rdb-archiver/internal/config"
	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/mailbox"
	"github.com/raoulx24/rdb-archiver/internal/retention"
	"github.com/raoulx24/rdb-archiver/internal/watcher"
	"github.com/raoulx24/rdb-archiver/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configFile := "config/config.yaml"

	// Temporary fallback logger
	stdLog := log.New(os.Stdout, "", log.LstdFlags)

	// 1️⃣ Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		stdLog.Fatalf("failed to load config: %v", err)
	}

	// Logger
	logg := logging.NewSlogLogger(cfg.Logging.Level, cfg.Logging.Format)

	// Graceful shutdown
	go func(logg logging.Logger) {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logg.Info("shutting down...")
		cancel()
	}(logg)

	// Mailbox for snapshot jobs
	mb := mailbox.New[worker.Job]()

	// Retention engine (promotion + cleanup)
	ret := retention.New(logg)

	// Worker (snapshot writer + promotion + cleanup)
	mainWorker := worker.New(
		cfg.Destination,
		logg,
		ret,
		mb,
		nil, // fs.FS (nil = default)
	)

	// Watcher (detects snapshots and pushes into mailbox)
	watch := watcher.New(
		cfg.Source,
		logg,
		mb,
	)

	// Start worker loop
	go mainWorker.Start(ctx)

	// Start watcher loop
	go func() {
		err := watch.Start(ctx)
		if err != nil {
			logg.Error("failed to start watcher", "error", err)
			os.Exit(1)
		}
	}()

	// Hot reload on config.yaml change
	go func() {
		dir := filepath.Dir(configFile)
		base := filepath.Base(configFile)

		configWatcher, err := fsnotify.NewWatcher()
		if err != nil {
			logg.Error("failed to create configWatcher", "error", err)
			os.Exit(1)
		}
		defer configWatcher.Close()

		if err := configWatcher.Add(dir); err != nil {
			logg.Error("failed to watch config directory", "error", err, "dir", dir)
			os.Exit(1)
		}

		// Debounce channel
		resetCh := make(chan struct{}, 1)

		// Debounce goroutine
		go func() {
			var t *time.Timer
			for range resetCh {
				if t != nil {
					t.Stop()
				}
				t = time.AfterFunc(300*time.Millisecond, func() {
					logg.Debug("reloading config file")

					newCfg, err := config.Load(configFile)
					if err != nil {
						logg.Error("config reload failed", "error", err, "configFile", configFile)
						return
					}

					mainWorker.UpdateConfig(newCfg.Destination)
					watch.UpdateConfig(newCfg.Source)

					logg.Info("config reloaded")
				})
			}
		}()

		for {
			select {
			case ev, ok := <-configWatcher.Events:
				if !ok {
					return
				}

				// Only react to the file we care about
				if filepath.Base(ev.Name) != base {
					continue
				}

				// React to CREATE, WRITE, RENAME (covers ConfigMap updates)
				if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
					continue
				}

				// Trigger debounce
				select {
				case resetCh <- struct{}{}:
				default:
				}

			case err := <-configWatcher.Errors:
				logg.Error("configWatcher error", "error", err)
			}
		}
	}()

	<-ctx.Done()
	stdLog.Println("exit complete")
}
