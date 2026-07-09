package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// API routes.
	r.Get("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello, World!"})
	})

	// Serve the frontend. In prod this is the embedded React build; in dev
	// (built with -tags dev) Vite serves the frontend, so this is a stub.
	r.Handle("/*", frontendHandler())

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
