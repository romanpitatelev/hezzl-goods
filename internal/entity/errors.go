package entity

import "errors"

var (
	ErrEmptyName            = errors.New("invalid good name")
	ErrInvalidIDOrProjectID = errors.New("id and projectID must be positive")
	ErrGoodNotFound         = errors.New("good is not found in the database")
	ErrNegativePriority     = errors.New("priority must be positive")
	ErrSamePriority         = errors.New("new priority equals to current priority")
)
