package models

import (
	"time"

	"github.com/google/uuid"
)

type RunStatus string

const (
	RunStatusActive    RunStatus = "ACTIVE"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusAbandoned RunStatus = "ABANDONED"
)

type Run struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	RoomVersionID uuid.UUID
	Seed          int64
	Status        RunStatus
	StartedAt     time.Time
	CompletedAt   *time.Time
}

type RunActionType string

const (
	RunActionTypePlayer RunActionType = "player"
	RunActionTypeTick   RunActionType = "tick"
)

type RunAction struct {
	ID              uuid.UUID
	RunID           uuid.UUID
	Seq             int
	ActionType      RunActionType
	ActionKey       *string
	ClientRequestID *string
	AppliedAt       time.Time
}
