package tasks

import "unicode/utf8"

const maxTaskErrorBytes = 256 * 1024

func limitTaskError(value string) string {
	if len(value) <= maxTaskErrorBytes {
		return value
	}

	limit := maxTaskErrorBytes
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return value[:limit]
}
