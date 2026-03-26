package user

import "errors"

var (
	ErrEmailTaken = errors.New("email already taken")
	ErrNotFound   = errors.New("user not found")
)
