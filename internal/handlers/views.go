package handlers

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/stpotter16/go-template/internal/handlers/middleware"
	"github.com/stpotter16/go-template/internal/handlers/sessions"
	"github.com/stpotter16/go-template/internal/store"
	"github.com/stpotter16/go-template/internal/types"
)

type viewProps struct {
	CsrfToken  string
	CspNonce   string
	ActivePage string
}

//go:embed templates
var templateFS embed.FS

var errorTmpl = template.Must(
	template.New("base.html").ParseFS(
		templateFS,
		"templates/layouts/base.html",
		"templates/layouts/app.html",
		"templates/pages/error.html",
	))

func renderAppError(w http.ResponseWriter, r *http.Request, status int) {
	nonce, _ := middleware.NonceFromContext(r.Context())
	w.WriteHeader(status)
	props := struct {
		viewProps
		Status int
	}{
		viewProps: viewProps{CspNonce: nonce},
		Status:    status,
	}
	if err := errorTmpl.Execute(w, props); err != nil {
		log.Printf("renderAppError: failed to render error template: %v", err)
	}
}

func loginGet() http.HandlerFunc {
	t := template.Must(
		template.New("base.html").
			ParseFS(
				templateFS,
				"templates/layouts/base.html",
				"templates/pages/login.html",
			))
	return func(w http.ResponseWriter, r *http.Request) {
		nonce, err := extractCspNonceOnly(r)
		if err != nil {
			log.Printf("Could not extract csp nonce from ctx: %v", err)
			http.Error(w, "Could not construct session nonce", http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, viewProps{CspNonce: nonce}); err != nil {
			log.Printf("Could not create login page: %v", err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
		}
	}
}

func indexGet(s store.Store, sessionManager sessions.SessionManger) http.HandlerFunc {
	t := template.Must(
		template.New("base.html").
			ParseFS(
				templateFS,
				"templates/layouts/base.html",
				"templates/layouts/app.html",
				"templates/pages/index.html",
			))
	return func(w http.ResponseWriter, r *http.Request) {
		nonce, err := extractCspNonceOnly(r)
		if err != nil {
			log.Printf("Could not extract csp nonce from ctx: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		clicks, err := s.GetClicks(r.Context())
		if err != nil {
			log.Printf("indexGet: failed to load clicks: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		props := struct {
			viewProps
			HasClicks bool
			Clicks    []types.Click
		}{
			viewProps: viewProps{CspNonce: nonce, ActivePage: "dashboard"},
			HasClicks: len(clicks) > 0,
			Clicks:    clicks,
		}

		if err := t.Execute(w, props); err != nil {
			log.Printf("Could not create index page: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
		}
	}
}

func extractCspNonceOnly(r *http.Request) (string, error) {
	cspNonce, err := middleware.NonceFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return cspNonce, nil
}
