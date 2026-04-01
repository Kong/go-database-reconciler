package utils

import (
	"github.com/google/uuid"
)

// UUID will generate a random v4 unique identifier
func UUID() string {
	return uuid.NewString()
}

// IsValidUUID checks if the given string is a valid UUID.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
