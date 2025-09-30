package defaults

const (
	// DefaultTopK is the default number of items to return in top K queries.
	DefaultTopK = 5
)

// GetTopKLimit returns the limit value for top K queries, using default if not specified.
func GetTopKLimit(limit int32) int32 {
	if limit == 0 {
		return DefaultTopK
	}
	return limit
}
