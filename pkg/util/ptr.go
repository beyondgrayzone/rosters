package util

// Ptr shall be used when dealing with values that already exist
func Ptr[T any](v T) *T {
	return &v
}
