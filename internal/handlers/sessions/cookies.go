package sessions

import (
	"fmt"
	"net/http"

	"github.com/stpotter16/go-template/internal/cookies"
)

const SESSION_COOKIE = "X-COIN-SESSION"
const SESSION_COOKIE_TTL = 3600 * 24 * 30

func (s SessionManger) readSessionCookie(r *http.Request) (string, error) {
	return cookies.ReadSigned(r, SESSION_COOKIE, s.sessionHmacSecretKey)
}

func (s SessionManger) writeSessionCookie(w http.ResponseWriter, session Session) error {
	cookieVal := fmt.Sprintf("%d::%s", session.UserId, session.ID)
	cookie := http.Cookie{
		Name:     SESSION_COOKIE,
		Value:    cookieVal,
		Path:     "/",
		MaxAge:   SESSION_COOKIE_TTL,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookies.WriteSigned(w, cookie, s.sessionHmacSecretKey)
}

func (s SessionManger) deleteSessionCookie(w http.ResponseWriter) error {
	cookie := http.Cookie{
		Name:     SESSION_COOKIE,
		Value:    "deleted",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookies.WriteSigned(w, cookie, s.sessionHmacSecretKey)
}
