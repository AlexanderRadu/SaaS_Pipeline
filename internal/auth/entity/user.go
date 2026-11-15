package entity

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrWrongPassword     = errors.New("wrong password")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}
