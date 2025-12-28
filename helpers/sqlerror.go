package helpers

import "strings"

func IsDuplicateEntryError(err error) bool {
	return strings.Contains(err.Error(), "1062") ||
		strings.Contains(err.Error(), "Duplicate entry")
}
