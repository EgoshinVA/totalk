package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenInvalid       = errors.New("token is invalid or expired")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrForbidden          = errors.New("forbidden")
	ErrTaskNotFound       = errors.New("task not found")
)
