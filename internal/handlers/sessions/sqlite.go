package sessions

import (
	"context"
	"log"
	"time"
)

const CLEANUP_INTERVAL = 5 * time.Minute
const SESSION_TTL = 60 * time.Minute

func (s SessionManger) sessionCleanup() {
	ticker := time.NewTicker(CLEANUP_INTERVAL)

	for range ticker.C {
		if err := s.deleteExpiredSessions(); err != nil {
			log.Printf("Failed to delete expired sessions: %v", err)
		}
	}
}

func (s SessionManger) deleteExpiredSessions() error {
	delete := `
	DELETE FROM
		session
	WHERE
		expires_at <= datetime('now', 'localtime')
	`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.Exec(ctx, delete)

	return err
}

func (s SessionManger) readSession(ctx context.Context, key string) ([]byte, error) {
	query := `
	SELECT
		value
	FROM
		session
	WHERE
		session_key = ? AND
		expires_at >= datetime('now', 'localtime')
	`

	row := s.db.QueryRow(ctx, query, key)
	var serializedSession []byte
	if err := row.Scan(&serializedSession); err != nil {
		log.Printf("Session key '%s' is invalid", key)
		return nil, err
	}
	return serializedSession, nil
}

func (s SessionManger) insertSession(ctx context.Context, key string, session []byte) error {
	insert := `
	INSERT OR REPLACE INTO
		session
	(
		session_key,
		value,
		expires_at
	)
	VALUES (
		?,
		?,
		?
	)`
	expires_time := time.Now().Add(SESSION_TTL).Format(time.RFC3339)

	_, err := s.db.Exec(
		ctx,
		insert,
		key,
		session,
		expires_time,
	)

	return err
}

func (s SessionManger) deleteSession(ctx context.Context, sessionId string) error {
	delete := `
	DELETE FROM
		session
	WHERE
		session_key = ?
	`
	_, err := s.db.Exec(ctx, delete, sessionId)

	return err
}
