package handlers

import (
	"log"
	"net/http"

	"github.com/stpotter16/go-template/internal/store"
)

func postClicks(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.CreateClick(r.Context()); err != nil {
			log.Printf("clickPost: %v", err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	}
}
