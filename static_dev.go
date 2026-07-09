//go:build dev

package main

import "net/http"

// frontendHandler is a stub for dev builds. In development the Vite dev server
// (npm run dev on :5173) serves the frontend with HMR and proxies /api here,
// so the Go server never serves static files.
func frontendHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "dev mode: run `npm run dev` and use the Vite server on :5173", http.StatusNotFound)
	})
}
