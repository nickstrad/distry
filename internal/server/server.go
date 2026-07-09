package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(context.Context) error
}

type Server struct {
	pool     Pinger
	frontend http.Handler
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func New(pool Pinger, frontend http.Handler) *Server {
	return &Server{pool: pool, frontend: frontend}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/api/health", s.health)
	r.Handle("/*", s.frontend)

	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	if err := s.pool.Ping(ctx); err != nil {
		writeHealth(w, http.StatusServiceUnavailable, healthResponse{Status: "error", DB: "unreachable"})
		return
	}

	writeHealth(w, http.StatusOK, healthResponse{Status: "ok", DB: "ok"})
}

func writeHealth(w http.ResponseWriter, status int, body healthResponse) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
