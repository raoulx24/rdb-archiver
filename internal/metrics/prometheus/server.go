package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
)

type Server interface {
	Start(ctx context.Context) error
}

type promServer struct {
	cfg     Config
	logg    logging.Logger
	handler http.Handler
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

	srv := &http.Server{
		Addr:              addr,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	s.logg.Info("metrics server is starting", "addr", addr)

	err := srv.ListenAndServe()
	s.logg.Info("metrics server is stopped")
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil

}
