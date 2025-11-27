package domain

import "errors"

var (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict is returned when a resource already exists.
	ErrConflict = errors.New("resource already exists")

	// ErrInvalidInput is returned when the input data is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrInternal is returned when an unexpected error occurs.
	ErrInternal = errors.New("internal error")
)
