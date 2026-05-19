package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/stpotter16/dump/internal/store"
)

func consolidateGet(s store.Store, threshold float32) http.HandlerFunc {
	t := template.Must(
		template.New("base.html").ParseFS(
			templateFS,
			"templates/layouts/base.html",
			"templates/layouts/app.html",
			"templates/pages/consolidate.html",
		))
	return func(w http.ResponseWriter, r *http.Request) {
		baseProps, err := extractAuthViewProps(r, "consolidate")
		if err != nil {
			log.Printf("consolidateGet: could not extract view props: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		ideas, err := s.GetIdeas(r.Context())
		if err != nil {
			log.Printf("consolidateGet: failed to load ideas: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
			return
		}

		props := struct {
			viewProps
			Clusters []ideaCluster
		}{
			viewProps: baseProps,
			Clusters:  buildClusters(ideas, threshold),
		}
		if err := t.Execute(w, props); err != nil {
			log.Printf("consolidateGet: could not render page: %v", err)
			renderAppError(w, r, http.StatusInternalServerError)
		}
	}
}

func consolidatePost(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []int `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := s.DeleteIdeas(r.Context(), req.IDs); err != nil {
			log.Printf("consolidatePost: failed to delete ideas: %v", err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
