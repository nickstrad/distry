//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:frontend/dist
var distFS embed.FS

// frontendHandler serves the embedded React build, falling back to index.html
// so client-side routing works.
func frontendHandler() http.Handler {
	dist, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	fsys := http.FS(dist)
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, err := fsys.Open(r.URL.Path); err != nil {
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}
