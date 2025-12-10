package server

import (
	"log/slog"
	"net/http"
	"path/filepath"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.lg.Info("new request", slog.String("addr", r.RemoteAddr), slog.String("method", r.Method), slog.String("path", r.URL.Path))
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	fs := http.FileServer(http.Dir(s.Config.StaticDir))
	s.mux.HandleFunc("GET /{$}", s.handleIndexPage)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	s.mux.HandleFunc("GET /stream", s.handleWebsocket)
}

func (s *Server) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.Config.StaticDir, "index.html"))
}
