package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sidsharma96/SysEscape/internal/platform/db"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

// UserRepo defines operations on the users table.
type UserRepo interface {
	UpsertByGitHubID(ctx context.Context, githubID int64, username string, displayName string) (*models.User, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

// PostgresUserRepo implements UserRepo using pgx/v5.
type PostgresUserRepo struct {
	db db.DBTX
}

// NewPostgresUserRepo returns a new PostgresUserRepo.
func NewPostgresUserRepo(d db.DBTX) *PostgresUserRepo {
	return &PostgresUserRepo{db: d}
}

func (r *PostgresUserRepo) UpsertByGitHubID(ctx context.Context, githubID int64, username string, displayName string) (*models.User, error) {
	const q = `
		INSERT INTO users (github_id, github_username, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (github_id) DO UPDATE
			SET github_username = EXCLUDED.github_username,
			    display_name    = EXCLUDED.display_name,
			    updated_at      = now()
		RETURNING id, github_id, github_username, display_name, role, created_at, updated_at`

	var u models.User
	err := r.db.QueryRow(ctx, q, githubID, username, displayName).Scan(
		&u.ID, &u.GitHubID, &u.GitHubUsername, &u.DisplayName,
		&u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	const q = `
		SELECT id, github_id, github_username, display_name, role, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u models.User
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&u.ID, &u.GitHubID, &u.GitHubUsername, &u.DisplayName,
		&u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
