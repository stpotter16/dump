package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/stpotter16/go-template/internal/store"
	"github.com/stpotter16/go-template/internal/types"
)

func (s Store) GetUserByUsername(ctx context.Context, username string) (types.User, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, username, password, is_admin, created_time, last_modified_time FROM user WHERE username = ?`,
		username,
	)

	var u types.User
	var isAdmin int
	var createdTime, lastModifiedTime string

	err := row.Scan(&u.ID, &u.Username, &u.Password, &isAdmin, &createdTime, &lastModifiedTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.User{}, store.ErrUserNotFound
		}
		return types.User{}, err
	}

	u.IsAdmin = isAdmin != 0

	u.CreatedTime, err = parseTime(createdTime)
	if err != nil {
		return types.User{}, err
	}

	u.LastModifiedTime, err = parseTime(lastModifiedTime)
	if err != nil {
		return types.User{}, err
	}

	return u, nil
}

func (s Store) CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) error {
	adminVal := 0
	if isAdmin {
		adminVal = 1
	}
	now := formatTime(time.Now().UTC())

	_, err := s.db.Exec(ctx,
		`INSERT INTO user (username, password, is_admin, created_time, last_modified_time) VALUES (?, ?, ?, ?, ?)`,
		username, passwordHash, adminVal, now, now,
	)
	return err
}
