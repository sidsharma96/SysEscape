package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated user in the system.
type User struct {
	ID             uuid.UUID
	GitHubID       int64
	GitHubUsername string
	DisplayName    string
	Role           string // "USER" or "ADMIN"
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Session represents an active user session.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}
