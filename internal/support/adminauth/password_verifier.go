package adminauth

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type PasswordVerifier struct{}

func NewPasswordVerifier() *PasswordVerifier {
	return &PasswordVerifier{}
}

func (v *PasswordVerifier) Verify(storedHash string, rawPassword string) bool {
	stored := strings.TrimSpace(storedHash)
	raw := strings.TrimSpace(rawPassword)
	if stored == "" || raw == "" {
		return false
	}

	if isBcryptHash(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(raw)) == nil
	}

	// Bootstrap compatibility: if DB contains raw text, compare in constant time.
	return subtle.ConstantTimeCompare([]byte(stored), []byte(raw)) == 1
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") ||
		strings.HasPrefix(value, "$2b$") ||
		strings.HasPrefix(value, "$2y$")
}
