package service

import "errors"

var (
	ErrInvalidCredentials         = errors.New("invalid login or password")
	ErrLoginAlreadyTaken          = errors.New("login already taken")
	ErrInvalidToken               = errors.New("invalid or expired token")
	ErrInvalidOrderNumber         = errors.New("invalid order number")
	ErrOrderConflict              = errors.New("order already uploaded by another user")
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by this user") // для кода 200
)
