package sessions

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/stpotter16/go-template/internal/store/db"
)

type contextKey struct {
	name string
}

const SESSION_KEY = "session"
const SESSION_ENV_KEY = "COIN_SESSION_ENV_KEY"

type Session struct {
	ID        string
	UserId    int
	CsrfToken string
}

type SessionManger struct {
	db                   *db.DB
	sessionHmacSecretKey string
}

func New(db db.DB, getenv func(string) string) (SessionManger, error) {
	hmacSecret := getenv(SESSION_ENV_KEY)
	if hmacSecret == "" {
		return SessionManger{}, errors.New("could not locate HMAC secret key")
	}

	s := SessionManger{
		db:                   &db,
		sessionHmacSecretKey: hmacSecret,
	}

	go s.sessionCleanup()

	return s, nil
}

func (s SessionManger) CreateSession(w http.ResponseWriter, r *http.Request, userId int) error {
	sessionId := uuid.NewString()
	csrfToken, err := generateCsrfToken()
	if err != nil {
		log.Printf("Failed to generate session csrf token: %v", err)
		return err
	}
	session := Session{
		ID:        sessionId,
		UserId:    userId,
		CsrfToken: csrfToken,
	}

	if err := s.writeSessionCookie(w, session); err != nil {
		log.Printf("Failed to set session cookie: %v", err)
		return err
	}

	serializedSession, err := serializeSession(session)
	if err != nil {
		return err
	}

	if err := s.insertSession(r.Context(), session.ID, serializedSession); err != nil {
		log.Printf("Failed to save session %v: %v", session, err)
		return err
	}
	return nil
}

func (s SessionManger) DeleteSession(w http.ResponseWriter, r *http.Request) error {
	session, err := s.loadSession(r)
	if err != nil {
		return err
	}
	if err = s.deleteSession(r.Context(), session.ID); err != nil {
		log.Printf("Could not delete session %v: %v", session, err)
		return err
	}
	if err = s.deleteSessionCookie(w); err != nil {
		log.Printf("Could not delete session cookie: %v", err)
		return err
	}
	return nil
}

func (s SessionManger) PopulateSessionContext(r *http.Request) (context.Context, error) {
	session, err := s.loadSession(r)

	if err != nil {
		log.Printf("Unable to populate session context: %v", err)
		return nil, err
	}

	ctxKey := contextKey{SESSION_KEY}
	return context.WithValue(r.Context(), ctxKey, session), nil
}

func (s SessionManger) SessionFromContext(ctx context.Context) (Session, error) {
	ctxKey := contextKey{SESSION_KEY}
	session, okay := ctx.Value(ctxKey).(Session)
	if !okay {
		log.Printf("Unable to extract session from context")
		return Session{}, errors.New("no session info in context")
	}
	return session, nil
}

func (s SessionManger) loadSession(r *http.Request) (Session, error) {
	cookie, err := s.readSessionCookie(r)
	if err != nil {
		log.Printf("Failed to read session cookie: %v", err)
		return Session{}, err
	}

	cookieVals := strings.SplitN(cookie, "::", 2)
	if len(cookieVals) != 2 {
		log.Printf("Invalid cookie value: %s", cookie)
		return Session{}, errors.New("cookie is invalid")
	}
	cookieToken := cookieVals[1]

	serializedSession, err := s.readSession(r.Context(), cookieToken)
	if err != nil {
		log.Printf("Failed to load session data for session %s: %v", cookieToken, err)
		return Session{}, err
	}

	session, err := deserializeSession(serializedSession)
	if err != nil {
		return Session{}, err
	}

	if cookieToken != session.ID {
		return Session{}, errors.New("invalid session token")
	}

	return session, nil
}

func generateCsrfToken() (string, error) {
	randomReader := rand.Reader
	byteSlice := make([]byte, 16)
	if _, err := randomReader.Read(byteSlice); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(byteSlice)
	return token, nil
}
