//go:build dev

package web

import "net/http"

func FrontendHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "dev mode: run `npm run dev` and use the Vite server on :5173", http.StatusNotFound)
	})
}
