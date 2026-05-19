package handlers

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/stpotter16/dump/internal/handlers/middleware"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store"
	"github.com/stpotter16/dump/internal/types"
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

func indexGet() http.HandlerFunc {
	t := template.Must(
		template.New("base.html").
			ParseFS(
				templateFS,
				"templates/layouts/base.html",
				"templates/layouts/app.html",
				"templates/pages/new.html",
			))
	return func(w http.ResponseWriter, r *http.Request) {
		props, err := extractAuthViewProps(r, "new")
		if err != nil {
			log.Printf("indexGet: could not extract view props: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, props); err != nil {
			log.Printf("indexGet: could not render page: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
		}
	}
}

func reviewGet(s store.Store, threshold float32) http.HandlerFunc {
	t := template.Must(
		template.New("base.html").
			ParseFS(
				templateFS,
				"templates/layouts/base.html",
				"templates/layouts/app.html",
				"templates/pages/review.html",
			))
	return func(w http.ResponseWriter, r *http.Request) {
		baseProps, err := extractAuthViewProps(r, "review")
		if err != nil {
			log.Printf("reviewGet: could not extract view props: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		ideas, err := s.GetIdeas(r.Context())
		if err != nil {
			log.Printf("reviewGet: failed to load ideas: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		populateRelated(ideas, threshold)

		props := struct {
			viewProps
			Ideas []types.Idea
		}{
			viewProps: baseProps,
			Ideas:     ideas,
		}
		if err := t.Execute(w, props); err != nil {
			log.Printf("reviewGet: could not render page: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
		}
	}
}

func extractAuthViewProps(r *http.Request, activePage string) (viewProps, error) {
	nonce, err := middleware.NonceFromContext(r.Context())
	if err != nil {
		return viewProps{}, err
	}
	session, err := sessions.GetSessionFromContext(r.Context())
	if err != nil {
		return viewProps{}, err
	}
	return viewProps{
		CsrfToken:  session.CsrfToken,
		CspNonce:   nonce,
		ActivePage: activePage,
	}, nil
}

func extractCspNonceOnly(r *http.Request) (string, error) {
	cspNonce, err := middleware.NonceFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return cspNonce, nil
}
