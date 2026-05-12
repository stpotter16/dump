package authentication

import (
	"context"
	"errors"
	"log"

	"github.com/stpotter16/go-template/internal/auth"
	"github.com/stpotter16/go-template/internal/store"
	"github.com/stpotter16/go-template/internal/types"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Authenticator struct {
	store store.Store
}

func New(store store.Store) Authenticator {
	return Authenticator{store: store}
}

func (a Authenticator) AuthenticateUser(ctx context.Context, req types.LoginRequest) (int, error) {
	user, err := a.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			log.Printf("Authentication failed: user not found")
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	ok, err := auth.VerifyPassword(req.Password, user.Password)
	if err != nil {
		log.Printf("Authentication failed: password verification error: %v", err)
		return 0, ErrInvalidCredentials
	}

	if !ok {
		log.Printf("Authentication failed: invalid password for user %s", req.Username)
		return 0, ErrInvalidCredentials
	}

	log.Printf("Authentication succeeded for user %s (id=%d)", req.Username, user.ID)
	return user.ID, nil
}
