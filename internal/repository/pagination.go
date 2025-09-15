package repository

// Page represents a simple limit/offset window for listing operations.
// I keep it intentionally small; advanced filtering belongs to higher layers.
type Page struct {
	Limit  int
	Offset int
}

// PageResult carries a slice of items and the total count matching the query.
// I return the total so clients can compute pagination without an extra round trip.
type PageResult[T any] struct {
	Items []T
	Total int
}
