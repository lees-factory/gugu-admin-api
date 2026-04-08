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

type SessionStatusReason string

const (
	SessionStatusReasonActive  SessionStatusReason = "ACTIVE"
	SessionStatusReasonRevoked SessionStatusReason = "REVOKED"
	SessionStatusReasonRotated SessionStatusReason = "ROTATED"
	SessionStatusReasonReused  SessionStatusReason = "REUSED"
	SessionStatusReasonExpired SessionStatusReason = "EXPIRED"
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

func (s LoginSession) Status(now time.Time) Status {
	if s.StatusReason(now) == SessionStatusReasonActive {
		return StatusActive
	}
	return StatusInactive
}

func (s LoginSession) StatusReason(now time.Time) SessionStatusReason {
	if s.RevokedAt != nil {
		return SessionStatusReasonRevoked
	}
	if s.ReuseDetectedAt != nil {
		return SessionStatusReasonReused
	}
	if s.RotatedAt != nil {
		return SessionStatusReasonRotated
	}
	if !s.ExpiresAt.After(now) {
		return SessionStatusReasonExpired
	}
	return SessionStatusReasonActive
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

type SessionListFilter struct {
	UserID        string
	Status        Status
	Revoked       *bool
	ReuseDetected *bool
}
