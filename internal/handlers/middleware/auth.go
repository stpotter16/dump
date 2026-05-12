package middleware

import (
	"net/http"

	"github.com/stpotter16/go-template/internal/handlers/sessions"
)

func NewViewAuthenticationRequiredMiddleware(sessionManager sessions.SessionManger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := sessionManager.PopulateSessionContext(r)

			if err != nil {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewApiAuthenticationRequiredMiddleware(sessionManager sessions.SessionManger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := sessionManager.PopulateSessionContext(r)
			if err != nil {
				http.Error(w, "Unauthorized request", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
