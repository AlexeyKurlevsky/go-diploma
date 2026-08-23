package models

type User struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	PasswordHash []byte `json:"-"` // не выводим в JSON
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
