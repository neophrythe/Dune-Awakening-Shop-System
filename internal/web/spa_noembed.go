//go:build !embed

package web

import "net/http"

// embeddedSPA returns nil in non-embed builds; spaHandler then falls back to the
// on-disk dev assets (or API-only mode).
func embeddedSPA() http.Handler { return nil }
