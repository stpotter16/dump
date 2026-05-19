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
	cfg Config,
) {
	// Static
	mux.Handle("GET /static/", http.StripPrefix("/static/", serveStaticFiles()))

	// Views
	mux.HandleFunc("GET /login", loginGet())

	// Views that need authentication
	viewAuthRequired := middleware.NewViewAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("GET /{$}", viewAuthRequired(indexGet()))
	mux.Handle("GET /review", viewAuthRequired(reviewGet(store, cfg.SimilarityThreshold)))
	mux.Handle("GET /consolidate", viewAuthRequired(consolidateGet(store, cfg.SimilarityThreshold)))

	// Auth
	mux.HandleFunc("POST /login", loginPost(authenticator, sessionManager))

	// Session-authenticated API endpoints with CSRF verification
	apiAuthRequired := middleware.NewApiAuthenticationRequiredMiddleware(sessionManager)
	mux.Handle("POST /ideas", apiAuthRequired(postIdeas(store, embedder)))
	mux.Handle("DELETE /ideas/{id}", apiAuthRequired(deleteIdea(store)))
	mux.Handle("POST /consolidate", apiAuthRequired(consolidatePost(store)))
}
