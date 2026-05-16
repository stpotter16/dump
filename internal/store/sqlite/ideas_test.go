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

func TestCreateAndGetIdeas_EmbeddingRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	embedding := []float32{0.1, -0.5, 0.0, 0.99}

	id, err := s.CreateIdea(ctx, "test idea", embedding)
	if err != nil {
		t.Fatalf("CreateIdea: %v", err)
	}
	if got, want := id > 0, true; got != want {
		t.Errorf("CreateIdea id = %d, want > 0", id)
	}

	ideas, err := s.GetIdeas(ctx)
	if err != nil {
		t.Fatalf("GetIdeas: %v", err)
	}
	if got, want := len(ideas), 1; got != want {
		t.Fatalf("len(ideas) = %d, want %d", got, want)
	}

	idea := ideas[0]
	if got, want := idea.Text, "test idea"; got != want {
		t.Errorf("idea.Text = %q, want %q", got, want)
	}
	if got, want := len(idea.Embedding), len(embedding); got != want {
		t.Fatalf("len(embedding) = %d, want %d", got, want)
	}
	for i := range embedding {
		if got, want := idea.Embedding[i], embedding[i]; got != want {
			t.Errorf("embedding[%d] = %v, want %v", i, got, want)
		}
	}
}

func TestCreateIdea_NilEmbeddingRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if _, err := s.CreateIdea(ctx, "no embedding yet", nil); err != nil {
		t.Fatalf("CreateIdea with nil embedding: %v", err)
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
