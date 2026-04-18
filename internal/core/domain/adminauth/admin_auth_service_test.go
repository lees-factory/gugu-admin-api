package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestService_Login_Success(t *testing.T) {
	finder := &stubFinder{
		user: &AdminUser{
			ID:           "master",
			LoginID:      "master-login",
			PasswordHash: "secret-hash",
			Active:       true,
		},
	}
	writer := &stubWriter{}
	issuer := &stubIssuer{
		token:     "signed-token",
		expiresAt: time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
	}
	clock := stubClock{now: time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)}

	svc := NewService(
		finder,
		writer,
		stubPasswordVerifier{ok: true},
		issuer,
		clock,
	)

	result, err := svc.Login(context.Background(), "master", "1234")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.AdminID != "master" {
		t.Fatalf("result.AdminID = %q, want master", result.AdminID)
	}
	if result.LoginID != "master-login" {
		t.Fatalf("result.LoginID = %q, want master-login", result.LoginID)
	}
	if result.AccessToken != "signed-token" {
		t.Fatalf("result.AccessToken = %q, want signed-token", result.AccessToken)
	}
	if result.TokenType != "Bearer" {
		t.Fatalf("result.TokenType = %q, want Bearer", result.TokenType)
	}
	if !result.ExpiresAt.Equal(issuer.expiresAt) {
		t.Fatalf("result.ExpiresAt = %v, want %v", result.ExpiresAt, issuer.expiresAt)
	}
	if writer.lastLoginID != "master" {
		t.Fatalf("writer.lastLoginID = %q, want master", writer.lastLoginID)
	}
	if !writer.lastLoginAt.Equal(clock.now) {
		t.Fatalf("writer.lastLoginAt = %v, want %v", writer.lastLoginAt, clock.now)
	}
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	svc := NewService(
		&stubFinder{user: nil},
		&stubWriter{},
		stubPasswordVerifier{ok: true},
		&stubIssuer{},
		stubClock{now: time.Now()},
	)

	_, err := svc.Login(context.Background(), "missing", "pw")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_VerifyAccessToken_Success(t *testing.T) {
	svc := NewService(
		&stubFinder{
			user: &AdminUser{
				ID:      "master",
				LoginID: "master-login",
				Active:  true,
			},
		},
		&stubWriter{},
		stubPasswordVerifier{ok: true},
		&stubIssuer{
			verifyAdminID: "master",
		},
		stubClock{now: time.Now()},
	)

	adminID, err := svc.VerifyAccessToken(context.Background(), "token")
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if adminID != "master" {
		t.Fatalf("VerifyAccessToken() adminID = %q, want master", adminID)
	}
}

func TestService_VerifyAccessToken_InvalidToken(t *testing.T) {
	svc := NewService(
		&stubFinder{user: nil},
		&stubWriter{},
		stubPasswordVerifier{ok: true},
		&stubIssuer{verifyErr: errors.New("bad token")},
		stubClock{now: time.Now()},
	)

	_, err := svc.VerifyAccessToken(context.Background(), "bad")
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("VerifyAccessToken() error = %v, want ErrInvalidAccessToken", err)
	}
}

type stubFinder struct {
	user *AdminUser
	err  error
}

func (s *stubFinder) GetByID(context.Context, string) (*AdminUser, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func (s *stubFinder) GetByLoginID(context.Context, string) (*AdminUser, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

type stubWriter struct {
	lastLoginID string
	lastLoginAt time.Time
	err         error
}

func (s *stubWriter) UpdateLastLoginAt(_ context.Context, id string, at time.Time) error {
	if s.err != nil {
		return s.err
	}
	s.lastLoginID = id
	s.lastLoginAt = at
	return nil
}

type stubPasswordVerifier struct {
	ok bool
}

func (s stubPasswordVerifier) Verify(string, string) bool {
	return s.ok
}

type stubIssuer struct {
	token         string
	expiresAt     time.Time
	err           error
	verifyAdminID string
	verifyExpires time.Time
	verifyErr     error
}

func (s *stubIssuer) Issue(string, time.Time) (string, time.Time, error) {
	if s.err != nil {
		return "", time.Time{}, s.err
	}
	return s.token, s.expiresAt, nil
}

func (s *stubIssuer) Verify(string, time.Time) (string, time.Time, error) {
	if s.verifyErr != nil {
		return "", time.Time{}, s.verifyErr
	}
	return s.verifyAdminID, s.verifyExpires, nil
}

type stubClock struct {
	now time.Time
}

func (s stubClock) Now() time.Time {
	return s.now
}
