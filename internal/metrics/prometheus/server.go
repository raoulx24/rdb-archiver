package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server interface {
	Start(ctx context.Context) error
}

type promServer struct {
	cfg     Config
	handler http.Handler
}

func New(config Config, handler http.Handler) Server {
	if !config.Enabled {
		return &noopServer{}
	}
	return &promServer{
		cfg:     config,
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

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil

}
