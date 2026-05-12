package sqlite

import (
	"time"

	"github.com/stpotter16/go-template/internal/store/db"
)

const (
	timeFormat = time.RFC3339
)

type Store struct {
	db *db.DB
}

func New(db db.DB) (Store, error) {
	store := Store{db: &db}
	err := store.runMigrations()
	if err != nil {
		return Store{}, err
	}
	return store, nil
}

func formatTime(t time.Time) string {
	return t.Format(timeFormat)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
