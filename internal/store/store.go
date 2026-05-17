package store

import (
	"context"

	"github.com/stpotter16/dump/internal/types"
)

type Store interface {
	CreateIdea(ctx context.Context, text string, embedding []float32) (int, error)
	GetIdeas(ctx context.Context) ([]types.Idea, error)
}
