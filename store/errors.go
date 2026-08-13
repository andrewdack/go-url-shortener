package store

import "errors"

var (
	ErrUserIdMismatch = errors.New("user ID does not own short URL")
	ErrNotFound       = errors.New("short url not found")
	ErrAlreadyExists  = errors.New("short_url already exists")
)
