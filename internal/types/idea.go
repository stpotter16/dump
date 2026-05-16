package types

import "time"

type RelatedIdea struct {
	ID          int
	Text        string
	CreatedTime time.Time
}

type Idea struct {
	ID          int
	Text        string
	CreatedTime time.Time
	Related     []RelatedIdea
}
