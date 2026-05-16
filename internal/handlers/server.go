package handlers

import (
	"net/http"

	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/middleware"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store"
)

func NewServer(
	store store.Store,
	sessionManager sessions.SessionManger,
	authenticator authentication.Authenticator,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, store, sessionManager, authenticator)
	handler := middleware.CspMiddleware(mux)
	handler = middleware.LoggingWrapper(handler)
	return handler
}
