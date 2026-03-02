package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sidsharma96/SysEscape/internal/platform/db"
	"github.com/sidsharma96/SysEscape/internal/token"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrRequestInProgress   = errors.New("request in progress")
)

type RunRepo interface {
	CreateRun(ctx context.Context, userID, roomVersionID uuid.UUID, seed int64) (*models.Run, error)
}

type StartRunInput struct {
	ClientRequestID string
	UserID          uuid.UUID
	RoomSlug        string
	RoomVersionID   uuid.UUID
	Engine          string
}

type StartRunResult struct {
	RunID    uuid.UUID
	RunToken string
}

type MintRunTokenInput struct {
	UserID uuid.UUID
	RunID  uuid.UUID
	Engine string
	Now    time.Time
}

type TokenMinter interface {
	MintRunToken(in MintRunTokenInput) (string, error)
}

type StaticTokenMinter func(in MintRunTokenInput) (string, error)

func (m StaticTokenMinter) MintRunToken(in MintRunTokenInput) (string, error) {
	return m(in)
}

type JWTTokenMinter struct {
	secret string
	ttl    time.Duration
	nowFn  func() time.Time
}

func NewJWTTokenMinter(secret string, ttl time.Duration) *JWTTokenMinter {
	return &JWTTokenMinter{
		secret: secret,
		ttl:    ttl,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (m *JWTTokenMinter) MintRunToken(in MintRunTokenInput) (string, error) {
	now := in.Now
	if now.IsZero() {
		now = m.nowFn()
	}
	return token.MintRunToken(m.secret, token.MintRunTokenInput{
		UserID: in.UserID,
		RunID:  in.RunID,
		Engine: in.Engine,
		TTL:    m.ttl,
		Now:    now,
	})
}

type RunService struct {
	db          db.DBTX
	runRepo     RunRepo
	tokenMinter TokenMinter
	now         func() time.Time
	seedFn      func() int64
}

func NewRunService(d db.DBTX, runRepo RunRepo, tokenMinter TokenMinter) *RunService {
	return &RunService{
		db:          d,
		runRepo:     runRepo,
		tokenMinter: tokenMinter,
		now: func() time.Time {
			return time.Now().UTC()
		},
		seedFn: func() int64 {
			return time.Now().UTC().UnixNano()
		},
	}
}

func (s *RunService) StartRun(ctx context.Context, input StartRunInput) (*StartRunResult, error) {
	if err := validateStartRunInput(input); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil || s.runRepo == nil || s.tokenMinter == nil {
		return nil, fmt.Errorf("run service dependencies are not configured")
	}

	fp, err := startRunFingerprint(input.RoomSlug)
	if err != nil {
		return nil, err
	}
	key := startRunIdempotencyKey(input.UserID, input.ClientRequestID)

	if existing, err := s.lookupIdempotency(ctx, key, fp); err != nil || existing != nil {
		if err != nil {
			return nil, err
		}
		tokenStr, mintErr := s.mintToken(input.UserID, existing.RunID, input.Engine)
		if mintErr != nil {
			return nil, mintErr
		}
		return &StartRunResult{RunID: existing.RunID, RunToken: tokenStr}, nil
	}

	reserved, err := s.reserveIdempotency(ctx, key, input.UserID, fp)
	if err != nil {
		return nil, err
	}
	if !reserved {
		existing, lookupErr := s.lookupIdempotency(ctx, key, fp)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existing == nil {
			return nil, ErrRequestInProgress
		}
		tokenStr, mintErr := s.mintToken(input.UserID, existing.RunID, input.Engine)
		if mintErr != nil {
			return nil, mintErr
		}
		return &StartRunResult{RunID: existing.RunID, RunToken: tokenStr}, nil
	}

	completed := false
	defer func() {
		if completed {
			return
		}
		_, _ = s.db.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1 AND response_body IS NULL`, key)
	}()

	run, err := s.runRepo.CreateRun(ctx, input.UserID, input.RoomVersionID, s.seedFn())
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(startRunStoredResponse{RunID: run.ID.String()})
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(ctx, `UPDATE idempotency_keys SET response_body = $2 WHERE key = $1`, key, body); err != nil {
		return nil, err
	}
	completed = true

	tokenStr, err := s.mintToken(input.UserID, run.ID, input.Engine)
	if err != nil {
		return nil, err
	}

	return &StartRunResult{RunID: run.ID, RunToken: tokenStr}, nil
}

func (s *RunService) mintToken(userID, runID uuid.UUID, engine string) (string, error) {
	return s.tokenMinter.MintRunToken(MintRunTokenInput{
		UserID: userID,
		RunID:  runID,
		Engine: engine,
		Now:    s.now(),
	})
}

type startRunStoredResponse struct {
	RunID string `json:"runId"`
}

func (s *RunService) lookupIdempotency(ctx context.Context, key, fingerprint string) (*StartRunResult, error) {
	var storedFP string
	var body []byte
	err := s.db.QueryRow(ctx, `SELECT response_fingerprint, response_body FROM idempotency_keys WHERE key = $1`, key).Scan(&storedFP, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if storedFP != fingerprint {
		return nil, ErrIdempotencyConflict
	}
	if len(body) == 0 {
		return nil, ErrRequestInProgress
	}

	var resp startRunStoredResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(resp.RunID)
	if err != nil {
		return nil, err
	}
	return &StartRunResult{RunID: runID}, nil
}

func (s *RunService) reserveIdempotency(ctx context.Context, key string, userID uuid.UUID, fingerprint string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO idempotency_keys (key, user_id, response_fingerprint, response_body, expires_at)
		VALUES ($1, $2, $3, NULL, $4)
		ON CONFLICT (key) DO NOTHING`, key, userID, fingerprint, s.now().Add(24*time.Hour))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func startRunIdempotencyKey(userID uuid.UUID, clientRequestID string) string {
	return fmt.Sprintf("startRun:%s:%s", userID.String(), strings.TrimSpace(clientRequestID))
}

func startRunFingerprint(roomSlug string) (string, error) {
	payload := struct {
		RoomSlug string `json:"roomSlug"`
	}{
		RoomSlug: roomSlug,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validateStartRunInput(in StartRunInput) error {
	clientRequestUUID, err := uuid.Parse(strings.TrimSpace(in.ClientRequestID))
	if err != nil || clientRequestUUID.Version() != 4 {
		return fmt.Errorf("clientRequestId must be UUID v4")
	}
	if in.UserID == uuid.Nil {
		return fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(in.RoomSlug) == "" {
		return fmt.Errorf("roomSlug is required")
	}
	if in.RoomVersionID == uuid.Nil {
		return fmt.Errorf("roomVersionId is required")
	}
	if strings.TrimSpace(in.Engine) == "" {
		return fmt.Errorf("engine is required")
	}
	switch in.Engine {
	case token.EngineA, token.EngineB:
	default:
		return fmt.Errorf("engine must be %q or %q", token.EngineA, token.EngineB)
	}
	return nil
}
