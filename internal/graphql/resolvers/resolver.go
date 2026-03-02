package resolvers

import (
	"context"
	"time"

	catalogrepo "github.com/sidsharma96/SysEscape/internal/catalog/repo"
	engineasvc "github.com/sidsharma96/SysEscape/internal/engine/a/service"
	"github.com/sidsharma96/SysEscape/internal/graphql/generated"
	publishsvc "github.com/sidsharma96/SysEscape/internal/platform/publish"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

type PublishService interface {
	Publish(ctx context.Context, input publishsvc.PublishInput) (*models.RoomVersion, error)
}

type RunService interface {
	StartRun(ctx context.Context, input engineasvc.StartRunInput) (*engineasvc.StartRunResult, error)
}

// Resolver holds dependencies for GraphQL resolvers.
type Resolver struct {
	CatalogRepo    catalogrepo.RoomRepo
	PublishService PublishService
	RunService     RunService
}

func mapRoom(room models.RoomWithLatestVersion) *generated.Room {
	return &generated.Room{
		ID:            room.ID.String(),
		Slug:          room.Slug,
		Title:         room.Title,
		District:      room.District,
		Engine:        generated.RoomEngine(room.Engine),
		Difficulty:    generated.RoomDifficulty(room.Difficulty),
		Description:   room.Description,
		LatestVersion: mapRoomVersion(room.LatestVersion),
	}
}

func mapRoomVersion(version *models.RoomVersion) *generated.RoomVersion {
	if version == nil {
		return nil
	}
	return &generated.RoomVersion{
		ID:            version.ID.String(),
		VersionNumber: version.VersionNumber,
		Status:        generated.RoomVersionStatus(version.Status),
		Changelog:     version.Changelog,
		PublishedAt:   version.PublishedAt.UTC().Format(time.RFC3339),
	}
}
