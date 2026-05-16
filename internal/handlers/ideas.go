package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"

	"github.com/stpotter16/dump/internal/store"
	"github.com/stpotter16/dump/internal/types"
)

const similarityThreshold float32 = 0.5

func populateRelated(ideas []types.Idea) {
	for i := range ideas {
		if ideas[i].Embedding == nil {
			continue
		}
		for j := range ideas {
			if i == j || ideas[j].Embedding == nil {
				continue
			}
			if cosineSimilarity(ideas[i].Embedding, ideas[j].Embedding) >= similarityThreshold {
				ideas[i].Related = append(ideas[i].Related, types.RelatedIdea{
					ID:          ideas[j].ID,
					Text:        ideas[j].Text,
					CreatedTime: ideas[j].CreatedTime,
				})
			}
		}
	}
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	denom := float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))
	return dot / denom
}

func postIdeas(s store.Store, embedder Embedder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "text is required", http.StatusBadRequest)
			return
		}

		embedding, err := embedder.Embed(r.Context(), req.Text)
		if err != nil {
			log.Printf("postIdeas: failed to embed text: %v", err)
		}

		if _, err := s.CreateIdea(r.Context(), req.Text, embedding); err != nil {
			log.Printf("postIdeas: failed to create idea: %v", err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
