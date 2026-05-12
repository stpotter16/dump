package handlers

import (
	"errors"
	"net/http"

	"github.com/stpotter16/go-template/internal/handlers/authentication"
	"github.com/stpotter16/go-template/internal/handlers/sessions"
	"github.com/stpotter16/go-template/internal/parse"
)

func loginPost(authenticator authentication.Authenticator, sessionManager sessions.SessionManger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parse.ParseLoginPost(r)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		userId, err := authenticator.AuthenticateUser(r.Context(), req)
		if err != nil {
			if errors.Is(err, authentication.ErrInvalidCredentials) {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := sessionManager.CreateSession(w, r, userId); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
