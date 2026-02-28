package models

import (
	"time"

	"github.com/google/uuid"
)

// Room represents an escape room in the catalog.
type Room struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	District    string
	Engine      string // "A" or "B"
	Difficulty  string // "L0", "L1", "L2", "L3"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoomVersion represents a published version of a room's content.
type RoomVersion struct {
	ID            uuid.UUID
	RoomID        uuid.UUID
	VersionNumber int
	Status        string // "PUBLISHED", "DEPRECATED", "DISABLED"
	BundleHash    *string
	Changelog     string
	PublishedAt   time.Time
}

// RoomWithLatestVersion pairs a room with its latest published version.
type RoomWithLatestVersion struct {
	Room
	LatestVersion *RoomVersion
}
