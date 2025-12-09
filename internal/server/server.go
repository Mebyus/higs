package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	Config Config

	hs http.Server

	mux *http.ServeMux

	lg *slog.Logger
}

func (s *Server) Run(ctx context.Context, lg *slog.Logger) error {
	s.lg = lg
	s.mux = http.NewServeMux()
	s.setupRoutes()

	return s.listenAndServe(ctx)
}

func (s *Server) listenAndServe(ctx context.Context) error {
	s.hs = http.Server{
		Addr: fmt.Sprintf(":%d", s.Config.Port),

		Handler: s,

		MaxHeaderBytes: 1 << 16,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,

		ReadHeaderTimeout: 2 * time.Second,

		DisableGeneralOptionsHandler: true,
	}

	go s.watchContextAndShutdown(ctx)
	err := s.hs.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) watchContextAndShutdown(ctx context.Context) {
	<-ctx.Done()

	// TODO: return this error via channel
	err := s.shutdown()
	if err != nil {
		s.lg.Error("shutdown", slog.String("error", err.Error()))
	}
}

func (s *Server) shutdown() error {
	const timeout = 2 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.hs.Shutdown(ctx)
}
