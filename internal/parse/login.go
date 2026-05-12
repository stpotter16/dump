package parse

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/stpotter16/go-template/internal/types"
)

func ParseLoginPost(r *http.Request) (types.LoginRequest, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return types.LoginRequest{}, err
	}

	if body.Username == "" || body.Password == "" {
		return types.LoginRequest{}, errors.New("username and password are required")
	}

	return types.LoginRequest{
		Username: body.Username,
		Password: body.Password,
	}, nil
}
