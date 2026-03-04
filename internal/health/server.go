package health

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/raoulx24/rdb-archiver/internal/snapshotwatcher"
)

type Server struct {
	cfg     Config
	watcher *snapshotwatcher.Watcher
	srv     *http.Server
	mu      sync.RWMutex
}

func New(config Config, watcher *snapshotwatcher.Watcher) *Server {
	return &Server{cfg: config, watcher: watcher}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	addr := ":" + strconv.FormatUint(uint64(s.cfg.port), 10)
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
	}()

	return s.srv.ListenAndServe()
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	watcher := s.watcher
	s.mu.RUnlock()

	if watcher.IsAlive(60 * time.Second) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "watcher not alive", http.StatusServiceUnavailable)
}
