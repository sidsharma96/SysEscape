package publish

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sidsharma96/SysEscape/internal/testutil"
)

type mockBundleStore struct {
	exists bool
	err    error
}

func (m *mockBundleStore) Upload(ctx context.Context, hash string, data io.Reader, size int64) error {
	return nil
}

func (m *mockBundleStore) Download(ctx context.Context, hash string) (io.ReadCloser, error) {
	return nil, io.EOF
}

func (m *mockBundleStore) Exists(ctx context.Context, hash string) (bool, error) {
	return m.exists, m.err
}

func TestPublishService_HappyPath(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, tx, "happy-user")
	seedRoom(t, ctx, tx, "happy-room")

	svc := NewPublishService(tx, &mockBundleStore{exists: true})
	clientRequestID := uuid.NewString()
	got, err := svc.Publish(ctx, PublishInput{
		ClientRequestID:  clientRequestID,
		UserID:           userID,
		RoomSlug:         "happy-room",
		Version:          1,
		Changelog:        "initial",
		BundleHashSha256: hashOf('a'),
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if got == nil || got.VersionNumber != 1 {
		t.Fatalf("Publish() returned unexpected value: %+v", got)
	}
}

func TestPublishService_IdempotentRetry(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, tx, "idempotent-user")
	seedRoom(t, ctx, tx, "idempotent-room")
	svc := NewPublishService(tx, &mockBundleStore{exists: true})
	in := PublishInput{ClientRequestID: uuid.NewString(), UserID: userID, RoomSlug: "idempotent-room", Version: 2, Changelog: "v2", BundleHashSha256: hashOf('b')}

	first, err := svc.Publish(ctx, in)
	if err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	second, err := svc.Publish(ctx, in)
	if err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent retry returned different IDs: %s vs %s", first.ID, second.ID)
	}
}

func TestPublishService_FingerprintConflict(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, tx, "conflict-user")
	seedRoom(t, ctx, tx, "conflict-room")
	svc := NewPublishService(tx, &mockBundleStore{exists: true})
	clientRequestID := uuid.NewString()

	_, err := svc.Publish(ctx, PublishInput{ClientRequestID: clientRequestID, UserID: userID, RoomSlug: "conflict-room", Version: 1, Changelog: "v1", BundleHashSha256: hashOf('c')})
	if err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}

	_, err = svc.Publish(ctx, PublishInput{ClientRequestID: clientRequestID, UserID: userID, RoomSlug: "conflict-room", Version: 1, Changelog: "v1", BundleHashSha256: hashOf('d')})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestPublishService_BundleNotFound(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, tx, "bundle-missing-user")
	seedRoom(t, ctx, tx, "bundle-missing-room")
	svc := NewPublishService(tx, &mockBundleStore{exists: false})

	_, err := svc.Publish(ctx, PublishInput{ClientRequestID: uuid.NewString(), UserID: userID, RoomSlug: "bundle-missing-room", Version: 1, Changelog: "v1", BundleHashSha256: hashOf('e')})
	if !errors.Is(err, ErrBundleNotFound) {
		t.Fatalf("error = %v, want ErrBundleNotFound", err)
	}
}

func TestPublishService_BundleNotFound_AllowsSystemAdminZeroUUID(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	seedSystemAdminUser(t, ctx, tx)
	seedRoom(t, ctx, tx, "bundle-missing-system-admin-room")
	svc := NewPublishService(tx, &mockBundleStore{exists: false})

	_, err := svc.Publish(ctx, PublishInput{
		ClientRequestID:  uuid.NewString(),
		UserID:           uuid.Nil,
		RoomSlug:         "bundle-missing-system-admin-room",
		Version:          1,
		Changelog:        "v1",
		BundleHashSha256: hashOf('g'),
	})
	if !errors.Is(err, ErrBundleNotFound) {
		t.Fatalf("error = %v, want ErrBundleNotFound", err)
	}
}

func TestPublishService_ActivateSetsPointer(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, tx, "activate-user")
	seedRoom(t, ctx, tx, "activate-room")
	svc := NewPublishService(tx, &mockBundleStore{exists: true})

	rv, err := svc.Publish(ctx, PublishInput{ClientRequestID: uuid.NewString(), UserID: userID, RoomSlug: "activate-room", Version: 3, Changelog: "v3", BundleHashSha256: hashOf('f'), Activate: true})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	var activeID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT active_room_version_id FROM rooms WHERE slug = $1`, "activate-room").Scan(&activeID)
	if err != nil {
		t.Fatalf("query active_room_version_id: %v", err)
	}
	if activeID == nil || *activeID != rv.ID {
		t.Fatalf("active_room_version_id = %v, want %v", activeID, rv.ID)
	}
}

func seedUser(t *testing.T, ctx context.Context, tx pgx.Tx, username string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, github_id, github_username, display_name, role)
		VALUES ($1, $2, $3, $4, 'ADMIN')`, id, int64(100000+len(username)), username, username)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedSystemAdminUser(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, github_id, github_username, display_name, role)
		VALUES ($1, 0, 'system-roomctl', 'System (roomctl)', 'ADMIN')
		ON CONFLICT (id) DO NOTHING`, uuid.Nil)
	if err != nil {
		t.Fatalf("seed system admin user: %v", err)
	}
}

func seedRoom(t *testing.T, ctx context.Context, tx pgx.Tx, slug string) {
	t.Helper()
	_, err := tx.Exec(ctx, `
		INSERT INTO rooms (slug, title, district, engine, difficulty, description)
		VALUES ($1, 'Room', 'District', 'A', 'L1', 'desc')`, slug)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}
}

func hashOf(ch byte) string { return strings.Repeat(string(ch), 64) }
