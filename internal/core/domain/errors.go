/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package domain

import "errors"

var (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict is returned when a resource already exists.
	ErrConflict = errors.New("resource already exists")

	// ErrInvalidInput is returned when the input data is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrPassportAlreadyPublished is returned when trying to publish a passport that is already published.
	ErrPassportAlreadyPublished = errors.New("passport already published")

	// ErrInternal is returned when an unexpected error occurs.
	ErrInternal = errors.New("internal error")
)
