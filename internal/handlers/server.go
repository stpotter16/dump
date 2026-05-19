package handlers

import (
	"context"
	"net/http"

	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/middleware"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Config struct {
	SimilarityThreshold float32
}

func NewServer(
	store store.Store,
	embedder Embedder,
	sessionManager sessions.SessionManger,
	authenticator authentication.Authenticator,
	cfg Config,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, store, embedder, sessionManager, authenticator, cfg)
	handler := middleware.CspMiddleware(mux)
	handler = middleware.LoggingWrapper(handler)
	return handler
}
