package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"distry/internal/auth"
	"distry/internal/config"
	"distry/internal/db"
	"distry/internal/problems"
	"distry/internal/server"
	"distry/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := db.Up(ctx, pool); err != nil {
		log.Fatal(err)
	}

	problemRepo := problems.NewPGRepo(pool)
	problemFS := os.DirFS(problemDir())
	loadedProblems, err := problems.LoadDir(problemFS)
	if err != nil {
		log.Fatal(err)
	}
	if err := problems.Sync(ctx, problemRepo, loadedProblems); err != nil {
		log.Fatal(err)
	}
	log.Printf("synced %d problems", len(loadedProblems))

	authService := auth.NewService(auth.NewPGUserRepo(pool), auth.NewPGSessionRepo(pool))
	app := server.New(pool, authService, problemRepo, web.FrontendHandler())
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()

	log.Printf("listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func problemDir() string {
	if dir := strings.TrimSpace(os.Getenv("DISTRY_PROBLEMS_DIR")); dir != "" {
		return dir
	}
	return "problems"
}
