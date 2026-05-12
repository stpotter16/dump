package handlers

import (
	"net/http"

	"github.com/stpotter16/go-template/internal/handlers/authentication"
	"github.com/stpotter16/go-template/internal/handlers/middleware"
	"github.com/stpotter16/go-template/internal/handlers/sessions"
	"github.com/stpotter16/go-template/internal/store"
)

func addRoutes(
	mux *http.ServeMux,
	store store.Store,
	sessionManager sessions.SessionManger,
	authenticator authentication.Authenticator,
) {
	// Static
	mux.Handle("GET /static/", http.StripPrefix("/static/", serveStaticFiles()))

	// Views
	mux.HandleFunc("GET /login", loginGet())

	// Views that need authentication
	viewAuthRequired := middleware.NewViewAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("GET /{$}", viewAuthRequired(indexGet(store, sessionManager)))

	// Auth
	mux.HandleFunc("POST /login", loginPost(authenticator, sessionManager))

	// Session authenticated API endpoints
	apiAuthRequired := middleware.NewApiAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("POST /clicks", apiAuthRequired(postClicks(store)))
}
