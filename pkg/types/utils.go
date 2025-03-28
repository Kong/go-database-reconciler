package types

import (
	"fmt"
)

func errDuplicateEntity(entityType string, entityId string) error {
	return fmt.Errorf("error: %s with ID %s already exists", entityType, entityId)
}
