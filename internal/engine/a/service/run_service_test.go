package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	enginerepo "github.com/sidsharma96/SysEscape/internal/engine/a/repo"
	"github.com/sidsharma96/SysEscape/internal/testutil"
)

func TestRunService_StartRun_CreatePathPersistsRun(t *testing.T) {
	ctx, tx := setupRunServiceDB(t)
	userID := seedRunUser(t, ctx, tx, "run-create-user")
	roomVersionID := seedRunRoomVersion(t, ctx, tx, "run-create-room")

	tokenCalls := 0
	svc := NewRunService(tx, enginerepo.NewPostgresRunRepo(tx), StaticTokenMinter(func(in MintRunTokenInput) (string, error) {
		tokenCalls++
		return "token-created", nil
	}))
	svc.seedFn = func() int64 { return 4242 }

	got, err := svc.StartRun(ctx, StartRunInput{
		ClientRequestID: uuid.NewString(),
		UserID:          userID,
		RoomSlug:        "run-create-room",
		RoomVersionID:   roomVersionID,
		Engine:          "A",
	})
	if err != nil {
		t.Fatalf("StartRun() err=%v", err)
	}
	if got.RunToken != "token-created" || tokenCalls != 1 {
		t.Fatalf("result=%+v tokenCalls=%d", got, tokenCalls)
	}
}

func TestRunService_StartRun_IdempotencyReplayAndConflict(t *testing.T) {
	ctx, tx := setupRunServiceDB(t)
	userID := seedRunUser(t, ctx, tx, "run-idempotent-user")
	roomVersionID := seedRunRoomVersion(t, ctx, tx, "run-idempotent-room")

	tokenCalls := 0
	svc := NewRunService(tx, enginerepo.NewPostgresRunRepo(tx), StaticTokenMinter(func(in MintRunTokenInput) (string, error) {
		tokenCalls++
		return fmt.Sprintf("token-%d", tokenCalls), nil
	}))
	clientRequestID := uuid.NewString()

	first, err := svc.StartRun(ctx, StartRunInput{
		ClientRequestID: clientRequestID,
		UserID:          userID,
		RoomSlug:        "run-idempotent-room",
		RoomVersionID:   roomVersionID,
		Engine:          "A",
	})
	if err != nil {
		t.Fatalf("first StartRun() err=%v", err)
	}
	second, err := svc.StartRun(ctx, StartRunInput{
		ClientRequestID: clientRequestID,
		UserID:          userID,
		RoomSlug:        "run-idempotent-room",
		RoomVersionID:   roomVersionID,
		Engine:          "A",
	})
	if err != nil {
		t.Fatalf("second StartRun() err=%v", err)
	}

	if first.RunID != second.RunID || first.RunToken == second.RunToken || tokenCalls != 2 {
		t.Fatalf("first=%+v second=%+v tokenCalls=%d", first, second, tokenCalls)
	}
	_, err = svc.StartRun(ctx, StartRunInput{
		ClientRequestID: clientRequestID,
		UserID:          userID,
		RoomSlug:        "different-room",
		RoomVersionID:   roomVersionID,
		Engine:          "A",
	})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err=%v, want ErrIdempotencyConflict", err)
	}
}

func setupRunServiceDB(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx := context.Background()
	return ctx, testutil.TxForTest(t, testutil.TestPool(t))
}

func seedRunUser(t *testing.T, ctx context.Context, tx pgx.Tx, username string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := tx.Exec(ctx, `INSERT INTO users (id, github_id, github_username, display_name, role) VALUES ($1, $2, $3, $4, 'USER')`, id, int64(2000000+len(username)), username, username)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedRunRoomVersion(t *testing.T, ctx context.Context, tx pgx.Tx, slug string) uuid.UUID {
	t.Helper()
	var roomID, versionID uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO rooms (slug, title, district, engine, difficulty, description) VALUES ($1, 'Run Room', 'distributed-systems', 'A', 'L1', 'desc') RETURNING id`, slug).Scan(&roomID); err != nil {
		t.Fatalf("seed room: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO room_versions (room_id, version_number, status, changelog) VALUES ($1, 1, 'PUBLISHED', 'initial') RETURNING id`, roomID).Scan(&versionID); err != nil {
		t.Fatalf("seed room_version: %v", err)
	}
	return versionID
}
