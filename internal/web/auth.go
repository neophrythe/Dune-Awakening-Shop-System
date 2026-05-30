package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookie = "dune_shop_session"
const sessionTTL = 12 * time.Hour

// authenticator issues and verifies HMAC-signed session cookies. It is
// stateless: the cookie carries "user|expiryUnix|signature", so no server-side
// session store is needed.
type authenticator struct {
	user   string
	pass   string
	secret []byte
	now    func() time.Time
}

func newAuthenticator(user, pass, secret string) *authenticator {
	return &authenticator{user: user, pass: pass, secret: []byte(secret), now: time.Now}
}

// checkCredentials reports whether the supplied login is valid (constant-time).
func (a *authenticator) checkCredentials(user, pass string) bool {
	u := subtle.ConstantTimeCompare([]byte(user), []byte(a.user))
	p := subtle.ConstantTimeCompare([]byte(pass), []byte(a.pass))
	return u == 1 && p == 1
}

func (a *authenticator) sign(payload string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// issue builds a signed session token valid for sessionTTL.
func (a *authenticator) issue() string {
	exp := a.now().Add(sessionTTL).Unix()
	payload := fmt.Sprintf("%s|%d", a.user, exp)
	return payload + "|" + a.sign(payload)
}

// verify checks a session token's signature and expiry.
func (a *authenticator) verify(token string) bool {
	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return false
	}
	payload := parts[0] + "|" + parts[1]
	if subtle.ConstantTimeCompare([]byte(a.sign(payload)), []byte(parts[2])) != 1 {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || a.now().Unix() > exp {
		return false
	}
	return true
}

// setCookie writes the session cookie on login.
func (a *authenticator) setCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    a.issue(),
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1,
	})
}

// requireAuth wraps a handler, rejecting requests without a valid session.
func (a *authenticator) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || !a.verify(c.Value) {
			writeErr(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		next(w, r)
	}
}
