package publish

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
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrRequestInProgress   = errors.New("request in progress")
	ErrBundleNotFound      = errors.New("bundle not found in storage")
	ErrRoomNotFound        = errors.New("room not found")
)

type PublishInput struct {
	ClientRequestID  string
	UserID           uuid.UUID
	RoomSlug         string
	Version          int
	Changelog        string
	BundleHashSha256 string
	Activate         bool
}

type Service struct {
	db          db.DBTX
	bundleStore storage.BundleStore
}

func NewPublishService(d db.DBTX, b storage.BundleStore) *Service {
	return &Service{db: d, bundleStore: b}
}

func (s *Service) Publish(ctx context.Context, input PublishInput) (*models.RoomVersion, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}
	fingerprint, err := fingerprint(input.RoomSlug, input.Version, input.BundleHashSha256)
	if err != nil {
		return nil, err
	}
	key := "publishRoomVersion:" + input.ClientRequestID

	if existing, err := s.lookupIdempotency(ctx, key, fingerprint); err != nil || existing != nil {
		return existing, err
	}

	reserved, err := s.reserveIdempotency(ctx, key, input.UserID, fingerprint)
	if err != nil {
		return nil, err
	}
	if !reserved {
		return s.lookupIdempotency(ctx, key, fingerprint)
	}
	completed := false
	defer func() {
		if completed {
			return
		}
		_, _ = s.db.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1 AND response_body IS NULL`, key)
	}()

	exists, err := s.bundleStore.Exists(ctx, input.BundleHashSha256)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrBundleNotFound
	}

	var roomID uuid.UUID
	if err := s.db.QueryRow(ctx, `SELECT id FROM rooms WHERE slug = $1`, input.RoomSlug).Scan(&roomID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	bundleHash := input.BundleHashSha256
	rv := &models.RoomVersion{}
	err = s.db.QueryRow(ctx, `
		INSERT INTO room_versions (room_id, version_number, status, bundle_hash, changelog)
		VALUES ($1, $2, 'PUBLISHED', $3, $4)
		RETURNING id, room_id, version_number, status, bundle_hash, changelog, published_at`,
		roomID, input.Version, bundleHash, input.Changelog,
	).Scan(&rv.ID, &rv.RoomID, &rv.VersionNumber, &rv.Status, &rv.BundleHash, &rv.Changelog, &rv.PublishedAt)
	if err != nil {
		return nil, err
	}

	if input.Activate {
		if _, err := s.db.Exec(ctx, `UPDATE rooms SET active_room_version_id = $1 WHERE id = $2`, rv.ID, roomID); err != nil {
			return nil, err
		}
	}

	resp, err := json.Marshal(rv)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(ctx, `UPDATE idempotency_keys SET response_body = $2 WHERE key = $1`, key, resp); err != nil {
		return nil, err
	}
	completed = true

	return rv, nil
}

func (s *Service) lookupIdempotency(ctx context.Context, key, fp string) (*models.RoomVersion, error) {
	var storedFP string
	var body []byte
	err := s.db.QueryRow(ctx, `SELECT response_fingerprint, response_body FROM idempotency_keys WHERE key = $1`, key).Scan(&storedFP, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if storedFP != fp {
		return nil, ErrIdempotencyConflict
	}
	if len(body) == 0 {
		return nil, ErrRequestInProgress
	}
	var rv models.RoomVersion
	if err := json.Unmarshal(body, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key string, userID uuid.UUID, fp string) (bool, error) {
	ct := time.Now().UTC().Add(24 * time.Hour)
	tag, err := s.db.Exec(ctx, `
		INSERT INTO idempotency_keys (key, user_id, response_fingerprint, response_body, expires_at)
		VALUES ($1, $2, $3, NULL, $4)
		ON CONFLICT (key) DO NOTHING`, key, userID, fp, ct)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func validateInput(in PublishInput) error {
	clientRequestUUID, err := uuid.Parse(strings.TrimSpace(in.ClientRequestID))
	if err != nil {
		return fmt.Errorf("clientRequestId must be a valid UUID")
	}
	if clientRequestUUID.Version() != 4 {
		return fmt.Errorf("clientRequestId must be UUID v4")
	}

	switch {
	case strings.TrimSpace(in.RoomSlug) == "":
		return fmt.Errorf("roomSlug is required")
	case in.Version < 1:
		return fmt.Errorf("version must be positive")
	case strings.TrimSpace(in.Changelog) == "":
		return fmt.Errorf("changelog is required")
	case strings.TrimSpace(in.BundleHashSha256) == "":
		return fmt.Errorf("bundleHashSha256 is required")
	}
	return nil
}

func fingerprint(slug string, version int, hash string) (string, error) {
	payload := struct {
		RoomSlug         string `json:"roomSlug"`
		Version          int    `json:"version"`
		BundleHashSha256 string `json:"bundleHashSha256"`
	}{RoomSlug: slug, Version: version, BundleHashSha256: hash}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
