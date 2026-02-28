package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sidsharma96/SysEscape/internal/platform/db"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

// SessionRepo defines operations on the sessions table.
type SessionRepo interface {
	Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error)
	GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
}

// PostgresSessionRepo implements SessionRepo using pgx/v5.
type PostgresSessionRepo struct {
	db db.DBTX
}

// NewPostgresSessionRepo returns a new PostgresSessionRepo.
func NewPostgresSessionRepo(d db.DBTX) *PostgresSessionRepo {
	return &PostgresSessionRepo{db: d}
}

func (r *PostgresSessionRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
	const q = `
		INSERT INTO sessions (user_id, expires_at)
		VALUES ($1, $2)
		RETURNING id, user_id, expires_at, created_at`

	var s models.Session
	err := r.db.QueryRow(ctx, q, userID, expiresAt).Scan(
		&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSessionRepo) GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	const q = `
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE id = $1`

	var s models.Session
	err := r.db.QueryRow(ctx, q, sessionID).Scan(
		&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSessionRepo) Delete(ctx context.Context, sessionID uuid.UUID) error {
	const q = `DELETE FROM sessions WHERE id = $1`
	_, err := r.db.Exec(ctx, q, sessionID)
	return err
}
