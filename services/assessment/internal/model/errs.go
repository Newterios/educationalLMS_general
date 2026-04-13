package model

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrRateLimited        = errors.New("rate limited")
	ErrRemoteUnavailable  = errors.New("remote unavailable")
)
