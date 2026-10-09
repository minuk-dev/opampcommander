package model

import "fmt"

// CheckResourceVersion rejects missing and stale public mutation preconditions.
func CheckResourceVersion(expected, current int64) error {
	if expected <= 0 {
		return fmt.Errorf("%w: metadata.resourceVersion is required; re-read the resource", ErrInvalidArgument)
	}

	if expected != current {
		return fmt.Errorf("%w: resourceVersion changed; re-read the resource", ErrConflict)
	}

	return nil
}

// CheckResourceIdentity makes path identity authoritative, allowing omitted body identity.
func CheckResourceIdentity(namespace, name, bodyNamespace, bodyName string) error {
	if (bodyNamespace != "" && bodyNamespace != namespace) || (bodyName != "" && bodyName != name) {
		return fmt.Errorf("%w: body identity differs from request path", ErrInvalidArgument)
	}

	return nil
}
