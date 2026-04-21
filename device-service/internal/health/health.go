// Package health provides a minimal HTTP health server for device-service.
// The server returns 503 until SetReady is called, then 200. This allows
// Docker health checks to gate on actual readiness — cache warm complete and
// all connections established — rather than just process liveness.
package health

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
)

// healthPort is the port the health server listens on inside the container.
// Not configurable — nothing outside the container needs to reach it.
const healthPort = ":8081"

// Server is a minimal HTTP health server.
type Server struct {
	ready atomic.Bool
}

// NewServer constructs a Server. The server starts in the not-ready state.
func NewServer() *Server {
	return &Server{}
}

// SetReady marks the server as ready. Subsequent health check probes return 200.
// This method is safe for concurrent use.
func (s *Server) SetReady() {
	s.ready.Store(true)
}

// Run starts the HTTP server and blocks until ctx is cancelled, at which point
// it shuts down gracefully. Intended to be called in a goroutine.
func (s *Server) Run(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)

	srv := &http.Server{
		Addr:    healthPort,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("health server: %v", err)
		}
	}()

	log.Printf("health server: listening on %s", healthPort)

	<-ctx.Done()

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("health server: shutdown error: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
