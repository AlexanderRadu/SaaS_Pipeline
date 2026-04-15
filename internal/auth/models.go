package auth

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}
