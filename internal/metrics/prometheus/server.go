package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
)

type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	NeedsRestart(newCfg Config) bool
	NeedMetricsRebuild(newCfg Config) bool
	UpdateConfig(cfg Config)
}

type promServer struct {
	cfg     Config
	logg    logging.Logger
	handler http.Handler
	mu      sync.RWMutex
	srv     *http.Server
}

func New(config Config, log logging.Logger, handler http.Handler) Server {
	logg := log.With("pkg", "prometheus")
	if !config.Enabled {
		logg.Debug("creating no-op server", "function", "New")
		return &noopServer{}
	}
	logg.Debug("creating metrics server", "function", "New")
	return &promServer{
		cfg:     config,
		logg:    logg,
		handler: handler,
	}
}

func (s *promServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)

	s.mu.Lock()
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	srv := s.srv
	s.mu.Unlock()

	// graceful shutdown
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	s.logg.Info("metrics server is starting", "addr", addr)

	err := srv.ListenAndServe()
	s.logg.Info("metrics server is stopped")
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *promServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.srv.Shutdown(shutdownCtx)
	s.srv = nil
	return err
}
