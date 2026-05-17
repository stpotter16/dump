package parse

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/stpotter16/dump/internal/types"
)

func ParseLoginPost(r *http.Request) (types.LoginRequest, error) {
	body := struct {
		Passphrase string `json:"passphrase"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return types.LoginRequest{}, err
	}

	if body.Passphrase == "" {
		return types.LoginRequest{}, errors.New("passphrase is required")
	}

	return types.LoginRequest{
		Passphrase: body.Passphrase,
	}, nil
}
