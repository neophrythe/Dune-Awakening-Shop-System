package web

import (
	"testing"
	"time"
)

func TestSessionRoundTrip(t *testing.T) {
	a := newAuthenticator("admin", "secret", "hmac-key")
	tok := a.issue()
	if !a.verify(tok) {
		t.Fatal("freshly issued token should verify")
	}
}

func TestSessionTamperRejected(t *testing.T) {
	a := newAuthenticator("admin", "secret", "hmac-key")
	tok := a.issue()
	if a.verify(tok + "x") {
		t.Fatal("tampered token must not verify")
	}
	if a.verify("admin|9999999999|forged") {
		t.Fatal("forged signature must not verify")
	}
}

func TestSessionExpiry(t *testing.T) {
	base := time.Now()
	a := newAuthenticator("admin", "secret", "hmac-key")
	a.now = func() time.Time { return base }
	tok := a.issue()
	a.now = func() time.Time { return base.Add(sessionTTL + time.Minute) }
	if a.verify(tok) {
		t.Fatal("expired token must not verify")
	}
}

func TestCheckCredentials(t *testing.T) {
	a := newAuthenticator("admin", "s3cret", "k")
	if !a.checkCredentials("admin", "s3cret") {
		t.Fatal("valid credentials should pass")
	}
	if a.checkCredentials("admin", "wrong") || a.checkCredentials("root", "s3cret") {
		t.Fatal("invalid credentials must fail")
	}
}

// differentSecretInvalidates ensures a token signed with one secret is rejected
// by an authenticator with another secret.
func TestDifferentSecretInvalidates(t *testing.T) {
	a1 := newAuthenticator("admin", "p", "secret-one")
	a2 := newAuthenticator("admin", "p", "secret-two")
	if a2.verify(a1.issue()) {
		t.Fatal("token from a different secret must not verify")
	}
}
