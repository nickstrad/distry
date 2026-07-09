package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"distry/internal/auth"
	"distry/internal/problems"
	"distry/internal/solutions"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(context.Context) error
}

type Server struct {
	pool        Pinger
	auth        *auth.Service
	problemRepo problems.Repo
	solutions   *solutions.Service
	frontend    http.Handler
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func New(pool Pinger, authService *auth.Service, problemRepo problems.Repo, solutionService *solutions.Service, frontend http.Handler) *Server {
	return &Server{pool: pool, auth: authService, problemRepo: problemRepo, solutions: solutionService, frontend: frontend}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	authHandler := auth.NewHandler(s.auth)
	r.Get("/api/health", s.health)
	r.Post("/api/auth/signup", authHandler.SignUp)
	r.Post("/api/auth/login", authHandler.LogIn)
	r.Post("/api/auth/logout", authHandler.LogOut)
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(s.auth))
		r.Get("/api/me", authHandler.Me)
		r.Get("/api/problems/{slug}/solution", s.getSolution)
		r.Put("/api/problems/{slug}/solution", s.putSolution)
	})
	r.Get("/api/problems", s.listProblems)
	r.Get("/api/problems/{slug}", s.getProblem)
	r.Handle("/*", s.frontend)

	return r
}

func (s *Server) getSolution(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	solution, err := s.solutions.Get(r.Context(), user.ID, solutionSlug(r))
	if errors.Is(err, solutions.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "solution not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get solution"})
		return
	}
	writeJSON(w, http.StatusOK, solution)
}

func (s *Server) putSolution(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var req solutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	solution := solutions.Solution{
		UserID:      user.ID,
		ProblemSlug: solutionSlug(r),
		Files:       req.Files,
	}
	if err := s.solutions.Upsert(r.Context(), solution); err != nil {
		writeSolutionSaveError(w, err)
		return
	}
	saved, err := s.savedSolution(r.Context(), user.ID, solution)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load saved solution"})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

type solutionRequest struct {
	Files map[string]string `json:"files"`
}

func (s *Server) savedSolution(ctx context.Context, userID string, fallback solutions.Solution) (solutions.Solution, error) {
	saved, err := s.solutions.Get(ctx, userID, fallback.ProblemSlug)
	if errors.Is(err, solutions.ErrNotFound) {
		return fallback, nil
	}
	return saved, err
}

func currentUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	return user, ok
}

func solutionSlug(r *http.Request) string {
	return chi.URLParam(r, "slug")
}

func writeSolutionSaveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, problems.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "problem not found"})
	case errors.Is(err, solutions.ErrValidation):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid solution files"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save solution"})
	}
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
