package types

import (
	"fmt"
)

func errDuplicateEntity(entityType string, entityID string) error {
	return fmt.Errorf("error: %s with ID %s already exists", entityType, entityID)
}
