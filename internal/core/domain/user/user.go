package user

import "time"

type Plan string

const (
	PlanFree Plan = "FREE"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

type User struct {
	ID               string
	Email            string
	DisplayName      string
	Plan             Plan
	Status           Status
	EmailVerified    bool
	TrackedItemCount int64
	CreatedAt        time.Time
	LastLoginAt      *time.Time
	Sessions         []LoginSession
}

type LoginSession struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	TokenFamilyID    string
	ParentSessionID  *string
	UserAgent        string
	ClientIP         string
	DeviceName       string
	ExpiresAt        time.Time
	LastSeenAt       time.Time
	RotatedAt        *time.Time
	RevokedAt        *time.Time
	ReuseDetectedAt  *time.Time
	CreatedAt        time.Time
}

type ListFilter struct {
	Search   string
	Plan     Plan
	Status   Status
	Page     int32
	PageSize int32
}

type ListResult struct {
	TotalCount int64
	Users      []User
}
