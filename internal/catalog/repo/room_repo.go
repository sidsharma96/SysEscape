package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sidsharma96/SysEscape/internal/platform/db"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

// RoomFilter specifies optional filters for listing rooms.
type RoomFilter struct {
	Engine     *string
	Difficulty *string
	District   *string
}

// RoomRepo defines read operations on the rooms catalog.
type RoomRepo interface {
	List(ctx context.Context, filter RoomFilter) ([]models.RoomWithLatestVersion, error)
	GetBySlug(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error)
}

// PostgresRoomRepo implements RoomRepo using pgx/v5.
type PostgresRoomRepo struct {
	db db.DBTX
}

// NewPostgresRoomRepo returns a new PostgresRoomRepo.
func NewPostgresRoomRepo(d db.DBTX) *PostgresRoomRepo {
	return &PostgresRoomRepo{db: d}
}

// baseQuery is the shared join that pairs each room with its latest published version.
// Ends with WHERE true so callers can append AND clauses.
const baseQuery = `
	SELECT
		r.id, r.slug, r.title, r.district, r.engine, r.difficulty, r.description,
		r.created_at, r.updated_at, r.active_room_version_id,
		v.id, v.room_id, v.version_number, v.status, v.bundle_hash, v.changelog, v.published_at
	FROM rooms r
	LEFT JOIN room_versions v ON v.room_id = r.id
		AND v.version_number = (
			SELECT MAX(v2.version_number)
			FROM room_versions v2
			WHERE v2.room_id = r.id AND v2.status = 'PUBLISHED'
		)
	WHERE true`

func (r *PostgresRoomRepo) List(ctx context.Context, filter RoomFilter) ([]models.RoomWithLatestVersion, error) {
	query := baseQuery
	args := []any{}
	argN := 1

	if filter.Engine != nil {
		query += fmt.Sprintf(" AND r.engine = $%d", argN)
		args = append(args, *filter.Engine)
		argN++
	}
	if filter.Difficulty != nil {
		query += fmt.Sprintf(" AND r.difficulty = $%d", argN)
		args = append(args, *filter.Difficulty)
		argN++
	}
	if filter.District != nil {
		query += fmt.Sprintf(" AND r.district = $%d", argN)
		args = append(args, *filter.District)
		argN++
	}

	query += " ORDER BY r.title"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.RoomWithLatestVersion
	for rows.Next() {
		rw, err := scanRoomWithVersion(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rw)
	}
	return result, rows.Err()
}

func (r *PostgresRoomRepo) GetBySlug(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error) {
	query := baseQuery + " AND r.slug = $1"

	rows, err := r.db.Query(ctx, query, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	rw, err := scanRoomWithVersion(rows)
	if err != nil {
		return nil, err
	}
	return &rw, nil
}

func scanRoomWithVersion(rows pgx.Rows) (models.RoomWithLatestVersion, error) {
	var rw models.RoomWithLatestVersion
	var activeRoomVersionID *uuid.UUID
	var vID *uuid.UUID
	var vRoomID *uuid.UUID
	var vNum *int
	var vStatus *string
	var vBundleHash *string
	var vChangelog *string
	var vPublishedAt *time.Time

	err := rows.Scan(
		&rw.ID, &rw.Slug, &rw.Title, &rw.District, &rw.Engine, &rw.Difficulty, &rw.Description,
		&rw.CreatedAt, &rw.UpdatedAt, &activeRoomVersionID,
		&vID, &vRoomID, &vNum, &vStatus, &vBundleHash, &vChangelog, &vPublishedAt,
	)
	if err != nil {
		return rw, err
	}

	rw.ActiveRoomVersionID = activeRoomVersionID

	if vID != nil {
		rw.LatestVersion = &models.RoomVersion{
			ID:            *vID,
			RoomID:        *vRoomID,
			VersionNumber: *vNum,
			Status:        *vStatus,
			BundleHash:    vBundleHash,
			Changelog:     *vChangelog,
			PublishedAt:   *vPublishedAt,
		}
	}

	return rw, nil
}
