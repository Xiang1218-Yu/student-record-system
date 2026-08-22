package service

import "errors"

// Sentinel errors let services communicate outcome categories without depending
// on HTTP, so handlers can map a service error to the right status code via
// errors.Is. Existing services still return plain errors.New strings; new code
// wraps these sentinels so the handler can branch.
var (
	// ErrInvalidInput signals a bad request: missing/invalid fields.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound signals the referenced resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict signals the request violates a uniqueness or state
	// constraint (e.g. email already registered, delete blocked by active
	// courses).
	ErrConflict = errors.New("conflict")
)
