package errors

import "errors"

var (
	// ErrEmptyRepository is returned when an RPC is missing a repository as an argument
	ErrEmptyRepository = errors.New("empty Repository")
	// ErrInvalidRepository is returned when an RPC has an invalid repository as an argument
	ErrInvalidRepository = errors.New("invalid Repository")
)
