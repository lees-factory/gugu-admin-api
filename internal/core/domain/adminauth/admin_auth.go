package adminauth

import "time"

type AdminUser struct {
	ID           string
	LoginID      string
	PasswordHash string
	Active       bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type LoginResult struct {
	AdminID     string
	LoginID     string
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}
