package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid login or password")
	ErrLoginAlreadyTaken  = errors.New("login already taken")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
