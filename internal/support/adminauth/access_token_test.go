package adminauth

import (
	"testing"
	"time"
)

func TestHMACTokenIssuer_IssueAndVerify(t *testing.T) {
	issuer := NewHMACTokenIssuer("test-secret", 2*time.Hour)
	now := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)

	token, expiresAt, err := issuer.Issue("master", now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatalf("Issue() token must not be empty")
	}
	if !expiresAt.Equal(now.Add(2 * time.Hour)) {
		t.Fatalf("Issue() expiresAt = %v, want %v", expiresAt, now.Add(2*time.Hour))
	}

	subject, verifiedExpiresAt, err := issuer.Verify(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if subject != "master" {
		t.Fatalf("Verify() subject = %q, want master", subject)
	}
	if !verifiedExpiresAt.Equal(expiresAt) {
		t.Fatalf("Verify() expiresAt = %v, want %v", verifiedExpiresAt, expiresAt)
	}
}

func TestHMACTokenIssuer_VerifyExpired(t *testing.T) {
	issuer := NewHMACTokenIssuer("test-secret", 1*time.Hour)
	now := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)

	token, _, err := issuer.Issue("master", now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, _, err = issuer.Verify(token, now.Add(2*time.Hour))
	if err == nil {
		t.Fatalf("Verify() error must not be nil")
	}
	if err != ErrTokenExpired {
		t.Fatalf("Verify() error = %v, want %v", err, ErrTokenExpired)
	}
}
