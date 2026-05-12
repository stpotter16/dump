package middleware_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stpotter16/go-template/internal/handlers/middleware"
	"github.com/stpotter16/go-template/internal/handlers/sessions"
	"github.com/stpotter16/go-template/internal/store/db"
	"github.com/stpotter16/go-template/internal/store/sqlite"
)

func newTestSessionManager(t *testing.T) sessions.SessionManger {
	t.Helper()

	d, err := db.New(t.TempDir())
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}

	// Run migrations so the session table exists.
	if _, err := sqlite.New(d); err != nil {
		t.Fatalf("sqlite.New (migrations): %v", err)
	}

	sm, err := sessions.New(d, func(key string) string {
		if key == sessions.SESSION_ENV_KEY {
			return "test-hmac-secret-key"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("sessions.New: %v", err)
	}

	return sm
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestApiAuthMiddleware_NoCookie_Returns401(t *testing.T) {
	sm := newTestSessionManager(t)
	handler := middleware.NewApiAuthenticationRequiredMiddleware(sm)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestApiAuthMiddleware_InvalidCookie_Returns401(t *testing.T) {
	sm := newTestSessionManager(t)
	handler := middleware.NewApiAuthenticationRequiredMiddleware(sm)(okHandler())

	// The cookie value must survive base64 URL-decode (done by the cookies
	// package), then fail the HMAC length/signature check. Encoding a short
	// plaintext produces a decoded value shorter than sha256.Size (32 bytes),
	// which causes ReadSigned to return ErrInvalidValue immediately.
	garbageValue := base64.URLEncoding.EncodeToString([]byte("invalid::cookie"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessions.SESSION_COOKIE,
		Value: garbageValue,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestViewAuthMiddleware_NoCookie_RedirectsToLogin(t *testing.T) {
	sm := newTestSessionManager(t)
	handler := middleware.NewViewAuthenticationRequiredMiddleware(sm)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status 307, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %q", location)
	}
}
