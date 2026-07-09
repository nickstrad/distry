//go:build !dev

package web

import (
	"net/http"
	"path"
)

const distDir = "frontend/dist"

func FrontendHandler() http.Handler {
	root := http.Dir(distDir)
	fileServer := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assetExists(root, r.URL.Path) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func assetExists(root http.Dir, urlPath string) bool {
	if urlPath == "/" {
		return true
	}

	file, err := root.Open(path.Clean(urlPath))
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	return err == nil && !info.IsDir()
}
