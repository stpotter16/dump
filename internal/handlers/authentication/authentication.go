package authentication

import (
	"errors"
	"log"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Authenticator struct {
	passphrase string
}

func New(passphrase string) Authenticator {
	return Authenticator{passphrase: passphrase}
}

func (a Authenticator) Authenticate(passphrase string) error {
	if passphrase != a.passphrase {
		log.Printf("Authentication failed: incorrect passphrase")
		return ErrInvalidCredentials
	}
	return nil
}
