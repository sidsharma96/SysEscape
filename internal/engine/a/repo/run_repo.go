package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sidsharma96/SysEscape/internal/platform/db"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

var ErrActionConflict = errors.New("action conflict")

type AppendActionInput struct {
	RunID           uuid.UUID
	ActionType      models.RunActionType
	ActionKey       *string
	ClientRequestID *string
}

type PostgresRunRepo struct {
	db db.DBTX
}

func NewPostgresRunRepo(d db.DBTX) *PostgresRunRepo {
	return &PostgresRunRepo{db: d}
}

func (r *PostgresRunRepo) CreateRun(ctx context.Context, userID, roomVersionID uuid.UUID, seed int64) (*models.Run, error) {
	const q = `
		INSERT INTO runs (user_id, room_version_id, seed)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, room_version_id, seed, status, started_at, completed_at`

	var run models.Run
	err := r.db.QueryRow(ctx, q, userID, roomVersionID, seed).Scan(
		&run.ID, &run.UserID, &run.RoomVersionID, &run.Seed,
		&run.Status, &run.StartedAt, &run.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *PostgresRunRepo) GetRunByID(ctx context.Context, runID uuid.UUID) (*models.Run, error) {
	const q = `
		SELECT id, user_id, room_version_id, seed, status, started_at, completed_at
		FROM runs
		WHERE id = $1`

	var run models.Run
	err := r.db.QueryRow(ctx, q, runID).Scan(
		&run.ID, &run.UserID, &run.RoomVersionID, &run.Seed,
		&run.Status, &run.StartedAt, &run.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *PostgresRunRepo) AppendAction(ctx context.Context, input AppendActionInput) (*models.RunAction, error) {
	if input.ClientRequestID != nil {
		existing, err := r.getActionByClientRequestID(ctx, input.RunID, *input.ClientRequestID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.ActionType == input.ActionType && nullableStringEqual(existing.ActionKey, input.ActionKey) {
				return existing, nil
			}
			return nil, ErrActionConflict
		}
	}

	const q = `
		WITH locked_run AS (
			SELECT id
			FROM runs
			WHERE id = $1
			FOR UPDATE
		),
		next_seq AS (
			SELECT COALESCE(MAX(seq), 0) + 1 AS seq
			FROM run_actions
			WHERE run_id = $1
		)
		INSERT INTO run_actions (run_id, seq, action_type, action_key, client_request_id)
		SELECT $1, next_seq.seq, $2, $3, $4
		FROM next_seq, locked_run
		RETURNING id, run_id, seq, action_type, action_key, client_request_id, applied_at`

	var action models.RunAction
	err := r.db.QueryRow(ctx, q, input.RunID, input.ActionType, input.ActionKey, input.ClientRequestID).Scan(
		&action.ID, &action.RunID, &action.Seq, &action.ActionType,
		&action.ActionKey, &action.ClientRequestID, &action.AppliedAt,
	)
	if err == nil {
		return &action, nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "idx_run_actions_dedup" || input.ClientRequestID == nil {
		return nil, err
	}

	existing, lookupErr := r.getActionByClientRequestID(ctx, input.RunID, *input.ClientRequestID)
	if lookupErr != nil {
		return nil, lookupErr
	}
	if existing == nil {
		return nil, err
	}
	if existing.ActionType == input.ActionType && nullableStringEqual(existing.ActionKey, input.ActionKey) {
		return existing, nil
	}
	return nil, ErrActionConflict
}

func (r *PostgresRunRepo) ListActions(ctx context.Context, runID uuid.UUID) ([]models.RunAction, error) {
	const q = `
		SELECT id, run_id, seq, action_type, action_key, client_request_id, applied_at
		FROM run_actions
		WHERE run_id = $1
		ORDER BY seq ASC`

	rows, err := r.db.Query(ctx, q, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := make([]models.RunAction, 0)
	for rows.Next() {
		var action models.RunAction
		if err := rows.Scan(
			&action.ID, &action.RunID, &action.Seq, &action.ActionType,
			&action.ActionKey, &action.ClientRequestID, &action.AppliedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

func (r *PostgresRunRepo) getActionByClientRequestID(ctx context.Context, runID uuid.UUID, clientRequestID string) (*models.RunAction, error) {
	const q = `
		SELECT id, run_id, seq, action_type, action_key, client_request_id, applied_at
		FROM run_actions
		WHERE run_id = $1 AND client_request_id = $2`

	var action models.RunAction
	err := r.db.QueryRow(ctx, q, runID, clientRequestID).Scan(
		&action.ID, &action.RunID, &action.Seq, &action.ActionType,
		&action.ActionKey, &action.ClientRequestID, &action.AppliedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func nullableStringEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
