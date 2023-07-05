package conditions

import "fmt"

// UpdateDeletedConditionError returns an error indicating an object was unable to update
// the deleted condition.
func UpdateDeletedConditionError(err error) error {
	return fmt.Errorf("error updating deleted condition - %w", err)
}

// UpdateReconcilingConditionError returns an error indicating an object was unable to update
// the reconciling condition.
func UpdateReconcilingConditionError(err error) error {
	return fmt.Errorf("error updating reconciling condition - %w", err)
}
