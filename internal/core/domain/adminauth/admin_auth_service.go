package adminauth

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAccessToken = errors.New("invalid access token")
)

type PasswordVerifier interface {
	Verify(storedHash string, rawPassword string) bool
}

type AccessTokenIssuer interface {
	Issue(adminID string, now time.Time) (token string, expiresAt time.Time, err error)
	Verify(token string, now time.Time) (adminID string, expiresAt time.Time, err error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	finder           Finder
	writer           Writer
	passwordVerifier PasswordVerifier
	tokenIssuer      AccessTokenIssuer
	clock            Clock
}

func NewService(
	finder Finder,
	writer Writer,
	passwordVerifier PasswordVerifier,
	tokenIssuer AccessTokenIssuer,
	clock Clock,
) *Service {
	return &Service{
		finder:           finder,
		writer:           writer,
		passwordVerifier: passwordVerifier,
		tokenIssuer:      tokenIssuer,
		clock:            clock,
	}
}

func (s *Service) Login(ctx context.Context, id string, password string) (*LoginResult, error) {
	loginID := strings.TrimSpace(id)
	rawPassword := strings.TrimSpace(password)
	if loginID == "" || rawPassword == "" {
		return nil, ErrInvalidCredentials
	}

	adminUser, err := s.finder.GetByLoginID(ctx, loginID)
	if err != nil {
		return nil, err
	}
	if adminUser == nil || !adminUser.Active {
		return nil, ErrInvalidCredentials
	}
	if !s.passwordVerifier.Verify(adminUser.PasswordHash, rawPassword) {
		return nil, ErrInvalidCredentials
	}

	now := s.clock.Now()
	if err := s.writer.UpdateLastLoginAt(ctx, adminUser.ID, now); err != nil {
		return nil, err
	}

	token, expiresAt, err := s.tokenIssuer.Issue(adminUser.ID, now)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AdminID:     adminUser.ID,
		LoginID:     adminUser.LoginID,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) VerifyAccessToken(ctx context.Context, accessToken string) (string, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return "", ErrInvalidAccessToken
	}

	adminID, _, err := s.tokenIssuer.Verify(token, s.clock.Now())
	if err != nil {
		return "", ErrInvalidAccessToken
	}
	adminUser, err := s.finder.GetByID(ctx, adminID)
	if err != nil {
		return "", err
	}
	if adminUser == nil || !adminUser.Active {
		return "", ErrInvalidAccessToken
	}

	return adminUser.ID, nil
}
