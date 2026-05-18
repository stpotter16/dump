package sqlite_test

import (
	"context"
	"testing"

	"github.com/stpotter16/dump/internal/store/db"
	"github.com/stpotter16/dump/internal/store/sqlite"
)

func newTestStore(t *testing.T) sqlite.Store {
	t.Helper()
	d, err := db.New(t.TempDir())
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	s, err := sqlite.New(d)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	return s
}

func TestUpdateIdeaEmbedding_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	embedding := []float32{0.1, -0.5, 0.0, 0.99}

	id, err := s.CreateIdea(ctx, "test idea")
	if err != nil {
		t.Fatalf("CreateIdea: %v", err)
	}

	if err := s.UpdateIdeaEmbedding(ctx, id, embedding); err != nil {
		t.Fatalf("UpdateIdeaEmbedding: %v", err)
	}

	ideas, err := s.GetIdeas(ctx)
	if err != nil {
		t.Fatalf("GetIdeas: %v", err)
	}
	if got, want := len(ideas), 1; got != want {
		t.Fatalf("len(ideas) = %d, want %d", got, want)
	}

	if got, want := len(ideas[0].Embedding), len(embedding); got != want {
		t.Fatalf("len(embedding) = %d, want %d", got, want)
	}
	for i := range embedding {
		if got, want := ideas[0].Embedding[i], embedding[i]; got != want {
			t.Errorf("embedding[%d] = %v, want %v", i, got, want)
		}
	}
}

func TestCreateIdea_EmbeddingIsNilBeforeUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if _, err := s.CreateIdea(ctx, "no embedding yet"); err != nil {
		t.Fatalf("CreateIdea: %v", err)
	}

	ideas, err := s.GetIdeas(ctx)
	if err != nil {
		t.Fatalf("GetIdeas: %v", err)
	}
	if got, want := len(ideas), 1; got != want {
		t.Fatalf("len(ideas) = %d, want %d", got, want)
	}
	if ideas[0].Embedding != nil {
		t.Errorf("embedding = %v, want nil", ideas[0].Embedding)
	}
}
