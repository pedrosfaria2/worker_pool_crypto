package storage

import "errors"

var (
	ErrNotFound      = errors.New("record not found")
	ErrInvalidData   = errors.New("invalid data")
	ErrAlreadyExists = errors.New("record already exists")
)
