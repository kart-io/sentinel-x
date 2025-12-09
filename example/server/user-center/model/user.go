package model

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // Don't return password
	Role     string `json:"role"`
}

// Mock users database
var Users = map[string]*User{
	"admin": {
		ID:       "1",
		Username: "admin",
		Password: "password", // In real app, use hash
		Role:     "admin",
	},
	"user": {
		ID:       "2",
		Username: "user",
		Password: "password",
		Role:     "user",
	},
}
