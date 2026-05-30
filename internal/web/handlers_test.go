package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
)

// newTestServer builds a Server with no store (only auth-path tests use it, which
// never touch the store).
func newTestServer() *Server {
	return New(config.WebConfig{
		AdminUser: "admin", AdminPassword: "pw", SessionSecret: "k",
	}, nil, "Solari")
}

func TestLoginFlow(t *testing.T) {
	s := newTestServer()
	h := s.Handler()

	// Wrong password -> 401.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"user":"admin","password":"nope"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login code = %d, want 401", rec.Code)
	}

	// Correct password -> 200 + session cookie.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"user":"admin","password":"pw"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("good login code = %d, want 200", rec.Code)
	}
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value == "" {
		t.Fatal("expected session cookie")
	}
}

func TestProtectedRequiresAuth(t *testing.T) {
	s := newTestServer()
	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /api/stats = %d, want 401", rec.Code)
	}
}

func TestSessionEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/session", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("session code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"authenticated":false`) {
		t.Fatalf("expected authenticated:false, got %s", rec.Body.String())
	}
}
