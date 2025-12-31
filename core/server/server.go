package server

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	http *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		http: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
