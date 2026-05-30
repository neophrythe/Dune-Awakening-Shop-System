package web

import (
	"net/http"
	"os"
)

// spaHandler serves the dashboard single-page app. Production builds embed the
// compiled assets (see spa_embed.go, built with -tags embed). Without that tag
// it falls back to ./web/dist on disk for local development, or returns nil when
// no assets are present (API-only mode).
func spaHandler() http.Handler {
	if h := embeddedSPA(); h != nil {
		return h
	}
	const dist = "./web/dist"
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		return diskSPA(dist)
	}
	return nil
}

// diskSPA serves files from dir, falling back to index.html for client-side
// routes. It serves index.html directly (not via http.FileServer) to avoid the
// FileServer "/index.html -> ./" redirect loop.
func diskSPA(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if _, err := os.Stat(dir + r.URL.Path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, dir+"/index.html")
	})
}
