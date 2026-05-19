package handlers

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"slices"
	"strconv"

	"github.com/stpotter16/dump/internal/store"
	"github.com/stpotter16/dump/internal/types"
)

func populateRelated(ideas []types.Idea, threshold float32) {
	for i := range ideas {
		if ideas[i].Embedding == nil {
			continue
		}
		for j := range ideas {
			if i == j || ideas[j].Embedding == nil {
				continue
			}
			if cosineSimilarity(ideas[i].Embedding, ideas[j].Embedding) >= threshold {
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

type ideaCluster struct {
	Keep   types.Idea
	Delete []types.Idea
}

// buildClusters groups ideas into similarity clusters using union-find for
// transitive grouping (A~B and B~C puts all three in one cluster). Ideas are
// sorted newest-first by the store, so the smallest index in each group is
// the most recently added and is designated the keeper.
func buildClusters(ideas []types.Idea, threshold float32) []ideaCluster {
	n := len(ideas)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	for i := range ideas {
		if ideas[i].Embedding == nil {
			continue
		}
		for j := i + 1; j < n; j++ {
			if ideas[j].Embedding == nil {
				continue
			}
			if cosineSimilarity(ideas[i].Embedding, ideas[j].Embedding) >= threshold {
				rx, ry := find(i), find(j)
				if rx != ry {
					parent[rx] = ry
				}
			}
		}
	}

	groups := map[int][]int{}
	for i := range ideas {
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	var clusters []ideaCluster
	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}
		keepIdx := slices.Min(indices)
		c := ideaCluster{Keep: ideas[keepIdx]}
		for _, idx := range indices {
			if idx != keepIdx {
				c.Delete = append(c.Delete, ideas[idx])
			}
		}
		clusters = append(clusters, c)
	}
	return clusters
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

		id, err := s.CreateIdea(r.Context(), req.Text)
		if err != nil {
			log.Printf("postIdeas: failed to create idea: %v", err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
			return
		}

		go embedAndStore(s, embedder, id, req.Text)

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteIdea(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			http.Error(w, "Invalid idea ID", http.StatusBadRequest)
			return
		}
		if err := s.DeleteIdeas(r.Context(), []int{id}); err != nil {
			log.Printf("deleteIdea: failed to delete idea %d: %v", id, err)
			http.Error(w, "Server issue - try again later", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func embedAndStore(s store.Store, embedder Embedder, id int, text string) {
	embedding, err := embedder.Embed(context.Background(), text)
	if err != nil {
		log.Printf("embedAndStore: failed to embed idea %d: %v", id, err)
		return
	}
	if err := s.UpdateIdeaEmbedding(context.Background(), id, embedding); err != nil {
		log.Printf("embedAndStore: failed to store embedding for idea %d: %v", id, err)
	}
}
