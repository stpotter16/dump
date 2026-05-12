package cookies

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
)

var (
	ErrCookieNotFound = errors.New("cookie not found")
	ErrInvalidValue   = errors.New("invalid cookie value")
	ErrValueTooLong   = errors.New("cookie value too long")
)

func ReadSigned(r *http.Request, name string, secretKey string) (string, error) {
	signedValue, err := read(r, name)
	if err != nil {
		return "", err
	}

	if len(signedValue) < sha256.Size {
		return "", ErrInvalidValue
	}

	signature := signedValue[:sha256.Size]
	value := signedValue[sha256.Size:]

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(name))
	mac.Write([]byte(value))

	expectedSignature := mac.Sum(nil)

	if !hmac.Equal([]byte(signature), expectedSignature) {
		return "", ErrInvalidValue
	}

	return value, nil
}

func WriteSigned(w http.ResponseWriter, cookie http.Cookie, secretKey string) error {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(cookie.Name))
	mac.Write([]byte(cookie.Value))
	signature := mac.Sum(nil)
	cookie.Value = string(signature) + cookie.Value
	return write(w, cookie)
}

func read(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", ErrCookieNotFound
	}

	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidValue
	}

	return string(value), nil
}

func write(w http.ResponseWriter, cookie http.Cookie) error {
	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

	if len(cookie.String()) > 4096 {
		return ErrValueTooLong
	}

	http.SetCookie(w, &cookie)

	return nil
}
