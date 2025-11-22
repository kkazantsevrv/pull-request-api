package service

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrPrecondition = errors.New("precondition failed") // статус merged
)
