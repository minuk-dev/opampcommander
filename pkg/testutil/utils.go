package testutil

// P is a utility function that returns a pointer to the value passed in.
func P[T any](v T) *T {
	return &v
}
