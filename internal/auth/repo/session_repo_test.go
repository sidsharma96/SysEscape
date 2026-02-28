package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/auth/repo"
	"github.com/sidsharma96/SysEscape/internal/testutil"
)

func TestPostgresSessionRepo_Create_and_GetByID(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	userRepo := repo.NewPostgresUserRepo(tx)
	sessionRepo := repo.NewPostgresSessionRepo(tx)

	ctx := context.Background()

	user, err := userRepo.UpsertByGitHubID(ctx, 77777, "session-test", "Session Test")
	if err != nil {
		t.Fatalf("UpsertByGitHubID() error = %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
	session, err := sessionRepo.Create(ctx, user.ID, expiresAt)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.ID == uuid.Nil {
		t.Error("expected non-nil session ID")
	}
	if session.UserID != user.ID {
		t.Errorf("UserID = %s, want %s", session.UserID, user.ID)
	}
	if !session.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", session.ExpiresAt, expiresAt)
	}
	if session.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	got, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.ID != session.ID {
		t.Errorf("GetByID().ID = %s, want %s", got.ID, session.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("GetByID().UserID = %s, want %s", got.UserID, user.ID)
	}
}

func TestPostgresSessionRepo_Delete(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	userRepo := repo.NewPostgresUserRepo(tx)
	sessionRepo := repo.NewPostgresSessionRepo(tx)

	ctx := context.Background()

	user, err := userRepo.UpsertByGitHubID(ctx, 88888, "delete-test", "Delete Test")
	if err != nil {
		t.Fatalf("UpsertByGitHubID() error = %v", err)
	}

	session, err := sessionRepo.Create(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := sessionRepo.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID() after delete error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}
