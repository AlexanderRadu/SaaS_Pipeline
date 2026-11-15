package security

import "golang.org/x/crypto/bcrypt"

type PasswordManager struct{}

func NewPasswordManager() *PasswordManager {
	return &PasswordManager{}
}

func (p *PasswordManager) Hash(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(passwordHash), nil
}

func (p *PasswordManager) ValidatePassword(password string, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
