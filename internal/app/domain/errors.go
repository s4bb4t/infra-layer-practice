package domain

import "errors"

// Sentinel errors. Adapters must translate their driver-specific errors
// (pgx.ErrNoRows, mongo.ErrNoDocuments, duplicate key codes, ...) into these,
// so that use cases and transport can react without knowing the technology.
var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrConflict          = errors.New("version conflict")
	ErrInvalidOrder      = errors.New("invalid order")
	ErrInvalidTransition = errors.New("invalid status transition")
)
