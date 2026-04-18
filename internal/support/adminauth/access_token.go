package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type accessTokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type accessTokenPayload struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type HMACTokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewHMACTokenIssuer(secret string, ttl time.Duration) *HMACTokenIssuer {
	trimmedSecret := strings.TrimSpace(secret)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &HMACTokenIssuer{
		secret: []byte(trimmedSecret),
		ttl:    ttl,
	}
}

func (i *HMACTokenIssuer) Issue(adminID string, now time.Time) (string, time.Time, error) {
	subject := strings.TrimSpace(adminID)
	if subject == "" || len(i.secret) == 0 {
		return "", time.Time{}, ErrInvalidToken
	}

	exp := now.Add(i.ttl).UTC()
	payload := accessTokenPayload{
		Subject:   subject,
		IssuedAt:  now.UTC().Unix(),
		ExpiresAt: exp.Unix(),
	}

	headerPart, err := marshalAndEncode(accessTokenHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	})
	if err != nil {
		return "", time.Time{}, err
	}
	payloadPart, err := marshalAndEncode(payload)
	if err != nil {
		return "", time.Time{}, err
	}

	unsigned := headerPart + "." + payloadPart
	signature := i.sign(unsigned)
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	return token, exp, nil
}

func (i *HMACTokenIssuer) Verify(token string, now time.Time) (string, time.Time, error) {
	if len(i.secret) == 0 {
		return "", time.Time{}, ErrInvalidToken
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return "", time.Time{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSig := i.sign(unsigned)
	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return "", time.Time{}, ErrInvalidToken
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	var payload accessTokenPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return "", time.Time{}, ErrInvalidToken
	}

	subject := strings.TrimSpace(payload.Subject)
	if subject == "" || payload.ExpiresAt <= 0 {
		return "", time.Time{}, ErrInvalidToken
	}
	expiresAt := time.Unix(payload.ExpiresAt, 0).UTC()
	if !expiresAt.After(now.UTC()) {
		return "", time.Time{}, ErrTokenExpired
	}

	return subject, expiresAt, nil
}

func (i *HMACTokenIssuer) sign(value string) []byte {
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func marshalAndEncode(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
