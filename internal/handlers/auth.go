package handlers

import (
	"errors"
	"net/http"

	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/parse"
)

func loginPost(authenticator authentication.Authenticator, sessionManager sessions.SessionManger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parse.ParseLoginPost(r)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := authenticator.Authenticate(req.Passphrase); err != nil {
			if errors.Is(err, authentication.ErrInvalidCredentials) {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := sessionManager.CreateSession(w, r); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
