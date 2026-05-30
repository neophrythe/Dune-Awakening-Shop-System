//go:build embed

package web

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed all:dist
var embeddedDist embed.FS

// embeddedSPA serves the compiled dashboard assets baked into the binary
// (built with -tags embed). index.html is served for unknown paths so the SPA's
// client-side routing works; it is written via ServeContent to avoid
// http.FileServer's "/index.html -> ./" redirect loop.
func embeddedSPA() http.Handler {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		log.Printf("web: embedded dist malformed: %v", err)
		return nil
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if f, err := sub.Open(r.URL.Path[1:]); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		f, err := sub.Open("index.html")
		if err != nil {
			http.Error(w, "index.html missing", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		if rs, ok := f.(io.ReadSeeker); ok {
			http.ServeContent(w, r, "index.html", time.Time{}, rs)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, f)
	})
}
