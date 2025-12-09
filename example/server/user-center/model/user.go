package model

import "golang.org/x/crypto/bcrypt"

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Don't return password hash
	Role         string `json:"role"`
}

// CheckPassword verifies the provided password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HashPassword generates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// mustHash panics if hash generation fails (for init data only)
func mustHash(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

// Mock users database with bcrypt hashed passwords
// In production, these would come from a database
var Users = map[string]*User{
	"admin": {
		ID:           "1",
		Username:     "admin",
		PasswordHash: mustHash("password"), // bcrypt hash of "password"
		Role:         "admin",
	},
	"user": {
		ID:           "2",
		Username:     "user",
		PasswordHash: mustHash("password"), // bcrypt hash of "password"
		Role:         "user",
	},
}
