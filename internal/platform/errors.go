package platform

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("conflict")
	ErrInvalid     = errors.New("invalid argument")
	ErrUnavailable = errors.New("service unavailable")
)

// Is reports whether target is in err's error chain.
func Is(err, target error) bool { return errors.Is(err, target) }
