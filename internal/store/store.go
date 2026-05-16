package store

import (
	"context"
	"errors"

	"github.com/stpotter16/dump/internal/types"
)

var ErrUserNotFound = errors.New("user not found")

type Store interface {
	// Users
	GetUserByUsername(ctx context.Context, username string) (types.User, error)
	CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) error

	// Ideas
	CreateIdea(ctx context.Context, text string, embedding []float32) (int, error)
	GetIdeas(ctx context.Context) ([]types.Idea, error)
}
