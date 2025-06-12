package entity

import "errors"

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	ErrEmptyName            = errors.New("invalid good name")
)
