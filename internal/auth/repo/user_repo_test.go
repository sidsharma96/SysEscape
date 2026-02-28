package repo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/auth/repo"
	"github.com/sidsharma96/SysEscape/internal/testutil"
)

func TestPostgresUserRepo_UpsertByGitHubID_NewUser(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresUserRepo(tx)

	user, err := r.UpsertByGitHubID(context.Background(), 12345, "octocat", "Octo Cat")
	if err != nil {
		t.Fatalf("UpsertByGitHubID() error = %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if user.GitHubID != 12345 {
		t.Errorf("GitHubID = %d, want 12345", user.GitHubID)
	}
	if user.GitHubUsername != "octocat" {
		t.Errorf("GitHubUsername = %q, want %q", user.GitHubUsername, "octocat")
	}
	if user.DisplayName != "Octo Cat" {
		t.Errorf("DisplayName = %q, want %q", user.DisplayName, "Octo Cat")
	}
	if user.Role != "USER" {
		t.Errorf("Role = %q, want %q", user.Role, "USER")
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestPostgresUserRepo_UpsertByGitHubID_ExistingUser(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresUserRepo(tx)

	ctx := context.Background()

	first, err := r.UpsertByGitHubID(ctx, 99999, "old-name", "Old Name")
	if err != nil {
		t.Fatalf("first UpsertByGitHubID() error = %v", err)
	}

	second, err := r.UpsertByGitHubID(ctx, 99999, "new-name", "New Name")
	if err != nil {
		t.Fatalf("second UpsertByGitHubID() error = %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("ID changed: %s -> %s", first.ID, second.ID)
	}
	if second.GitHubUsername != "new-name" {
		t.Errorf("GitHubUsername = %q, want %q", second.GitHubUsername, "new-name")
	}
	if second.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q, want %q", second.DisplayName, "New Name")
	}
}

func TestPostgresUserRepo_GetByID_Found(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresUserRepo(tx)

	ctx := context.Background()

	created, err := r.UpsertByGitHubID(ctx, 55555, "get-test", "Get Test")
	if err != nil {
		t.Fatalf("UpsertByGitHubID() error = %v", err)
	}

	got, err := r.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %s, want %s", got.ID, created.ID)
	}
	if got.GitHubID != 55555 {
		t.Errorf("GitHubID = %d, want 55555", got.GitHubID)
	}
	if got.GitHubUsername != "get-test" {
		t.Errorf("GitHubUsername = %q, want %q", got.GitHubUsername, "get-test")
	}
	if got.Role != "USER" {
		t.Errorf("Role = %q, want %q", got.Role, "USER")
	}
}

func TestPostgresUserRepo_GetByID_NotFound(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresUserRepo(tx)

	user, err := r.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user for non-existent ID, got %+v", user)
	}
}
