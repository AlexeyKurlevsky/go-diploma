package models

import "github.com/google/uuid"

type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash []byte    `json:"-"` // не выводим в JSON
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
