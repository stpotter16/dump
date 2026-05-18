package store

import (
	"context"

	"github.com/stpotter16/dump/internal/types"
)

type Store interface {
	CreateIdea(ctx context.Context, text string) (int, error)
	UpdateIdeaEmbedding(ctx context.Context, id int, embedding []float32) error
	GetIdeas(ctx context.Context) ([]types.Idea, error)
}
