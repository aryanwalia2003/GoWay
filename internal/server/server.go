package server

import (
	"context"
	"net/http"
	"os"
	"time"
)

// Server wraps the http.Server for graceful shutdown.
type Server struct {
	addr    string
	handler http.Handler
	srv     *http.Server
	timeout time.Duration
	exitCh  chan struct{}
}

// NewServer initializes a new Server with a 15s default graceful shutdown.
func NewServer(addr string, handler http.Handler) *Server {
	return NewServerWithTimeout(addr, handler, 15*time.Second)
}

// NewServerWithTimeout initializes a server with a custom shutdown timeout.
func NewServerWithTimeout(a string, h http.Handler, t time.Duration) *Server {
	return &Server{
		addr:    a,
		handler: h,
		srv:     &http.Server{Addr: a, Handler: h},
		timeout: t,
		exitCh:  make(chan struct{}),
	}
}

// Addr returns the configured address.
func (s *Server) Addr() string {
	return s.addr
}

// Start listens and serves traffic, unblocking exitCh when done.
func (s *Server) Start() error {
	defer close(s.exitCh)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Signal triggers a graceful shutdown with the configured timeout.
func (s *Server) Signal(sig os.Signal) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
}

// Wait blocks until the server has completely exited.
func (s *Server) Wait() {
	<-s.exitCh
}
