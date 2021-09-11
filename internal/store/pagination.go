package store

type Pagination struct {
	MaxID   string
	MinID   string
	SinceID string
	Limit   int
}
