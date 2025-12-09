package server

import (
	"net/http"
	"path/filepath"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	fs := http.FileServer(http.Dir(s.Config.StaticDir))
	s.mux.HandleFunc("GET /{$}", s.indexHandler)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.Config.StaticDir, "index.html"))
}
