package domain

import "errors"

var (

	// ErrUserNotFound — пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// Avatar
	// ErrInvalidInput - неправильные вводимые данные
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrInternal     = errors.New("internal error")
)
