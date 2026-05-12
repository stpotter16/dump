package types

import "time"

type User struct {
	ID               int
	Username         string
	Password         string
	IsAdmin          bool
	CreatedTime      time.Time
	LastModifiedTime time.Time
}
