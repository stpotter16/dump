package handlers

import (
	"net/http"

	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/middleware"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store"
)

func addRoutes(
	mux *http.ServeMux,
	store store.Store,
	embedder Embedder,
	sessionManager sessions.SessionManger,
	authenticator authentication.Authenticator,
) {
	// Static
	mux.Handle("GET /static/", http.StripPrefix("/static/", serveStaticFiles()))

	// Views
	mux.HandleFunc("GET /login", loginGet())

	// Views that need authentication
	viewAuthRequired := middleware.NewViewAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("GET /{$}", viewAuthRequired(indexGet()))
	mux.Handle("GET /review", viewAuthRequired(reviewGet(store)))
	mux.Handle("GET /consolidate", viewAuthRequired(consolidateGet(store)))

	// Auth
	mux.HandleFunc("POST /login", loginPost(authenticator, sessionManager))

	// Session-authenticated API endpoints with CSRF verification
	apiAuthRequired := middleware.NewApiAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("POST /ideas", apiAuthRequired(postIdeas(store, embedder)))
	mux.Handle("POST /consolidate", apiAuthRequired(consolidatePost(store)))
}
