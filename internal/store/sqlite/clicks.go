package sqlite

import (
	"context"
	"time"

	"github.com/stpotter16/go-template/internal/types"
)

func (s Store) GetClicks(ctx context.Context) ([]types.Click, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, created_time from clicks`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clicks []types.Click

	for rows.Next() {
		var click types.Click
		var createdTime string
		if err := rows.Scan(&click.ID, &createdTime); err != nil {
			return nil, err
		}
		click.CreatedTime, err = parseTime(createdTime)
		if err != nil {
			return nil, err
		}
		clicks = append(clicks, click)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clicks, nil
}

func (s Store) CreateClick(ctx context.Context) (int, error) {
	now := formatTime(time.Now().UTC())
	result, err := s.db.Exec(ctx,
		`INSERT INTO clicks
			(created_time)
		VALUES ( ? )`,
		now,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}
