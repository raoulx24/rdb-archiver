package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/logging"
	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

type Server struct {
	cfg     Config
	logg    logging.Logger
	watcher *snapshotwatcher.Watcher
	srv     *http.Server
	mu      sync.RWMutex
}

func New(config Config, log logging.Logger, watcher *snapshotwatcher.Watcher) *Server {
	logg := log.With("pkg", "health")
	logg.Debug("creating health server", "function", "New")
	return &Server{
		cfg:     config,
		logg:    logg,
		watcher: watcher,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.mu.RUnlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", s.ready)
	mux.HandleFunc("/live", s.live)

	s.srv = &http.Server{Addr: addr, Handler: mux}

	// Shutdown goroutine
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutdownCtx)
		s.logg.Info("health server stopped")
	}()

	s.logg.Info("health server is starting", "addr", addr)

	err := s.srv.ListenAndServe()
	s.logg.Info("health server is stopped")
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	s.logg.Debug("health ready ok", "function", "ready")
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	watcher := s.watcher
	s.mu.RUnlock()

	if watcher.IsAlive(60 * time.Second) {
		w.WriteHeader(http.StatusOK)
		s.logg.Debug("health live ok", "function", "live")
		return
	}
	http.Error(w, "watcher not alive", http.StatusServiceUnavailable)
	s.logg.Debug("health live not ok", "function", "live")
}
