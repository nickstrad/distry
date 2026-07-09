package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"distry/internal/problems"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(context.Context) error
}

type Server struct {
	pool        Pinger
	problemRepo problems.Repo
	frontend    http.Handler
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func New(pool Pinger, problemRepo problems.Repo, frontend http.Handler) *Server {
	return &Server{pool: pool, problemRepo: problemRepo, frontend: frontend}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/api/health", s.health)
	r.Get("/api/problems", s.listProblems)
	r.Get("/api/problems/{slug}", s.getProblem)
	r.Handle("/*", s.frontend)

	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "error", DB: "unreachable"})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", DB: "ok"})
}

func (s *Server) listProblems(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.problemRepo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list problems"})
		return
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) getProblem(w http.ResponseWriter, r *http.Request) {
	problem, err := s.problemRepo.Get(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, problems.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "problem not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get problem"})
		return
	}
	writeJSON(w, http.StatusOK, problem)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
