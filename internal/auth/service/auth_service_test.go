package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

type mockUserRepo struct {
	upsertByGitHubIDFn func(ctx context.Context, githubID int64, username, displayName string) (*models.User, error)
	getByIDFn          func(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

func (m *mockUserRepo) UpsertByGitHubID(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
	if m.upsertByGitHubIDFn == nil {
		return nil, nil
	}
	return m.upsertByGitHubIDFn(ctx, githubID, username, displayName)
}

func (m *mockUserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, userID)
}

type mockSessionRepo struct {
	createFn  func(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error)
	getByIDFn func(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	deleteFn  func(ctx context.Context, sessionID uuid.UUID) error
}

func (m *mockSessionRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, expiresAt)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, sessionID)
}

func (m *mockSessionRepo) Delete(ctx context.Context, sessionID uuid.UUID) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(ctx, sessionID)
}

func TestAuthService_ExchangeCodeForUser_ValidCode(t *testing.T) {
	expectedUser := &models.User{
		ID:             uuid.New(),
		GitHubID:       42,
		GitHubUsername: "octocat",
		DisplayName:    "Octo Cat",
		Role:           "USER",
	}

	userRepo := &mockUserRepo{
		upsertByGitHubIDFn: func(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
			if githubID != 42 {
				t.Fatalf("githubID = %d, want %d", githubID, 42)
			}
			if username != "octocat" {
				t.Fatalf("username = %q, want %q", username, "octocat")
			}
			if displayName != "Octo Cat" {
				t.Fatalf("displayName = %q, want %q", displayName, "Octo Cat")
			}
			return expectedUser, nil
		},
	}

	githubClient := &MockGitHubClient{
		ExchangeCodeFunc: func(ctx context.Context, code string) (string, error) {
			if code != "valid" {
				t.Fatalf("code = %q, want %q", code, "valid")
			}
			return "token", nil
		},
		GetUserFunc: func(ctx context.Context, accessToken string) (GitHubUser, error) {
			if accessToken != "token" {
				t.Fatalf("accessToken = %q, want %q", accessToken, "token")
			}
			return GitHubUser{ID: 42, Login: "octocat", Name: "Octo Cat"}, nil
		},
	}

	svc := NewAuthService(userRepo, &mockSessionRepo{}, githubClient, OAuthConfig{})

	gotUser, err := svc.ExchangeCodeForUser(context.Background(), "valid")
	if err != nil {
		t.Fatalf("ExchangeCodeForUser() error = %v", err)
	}
	if gotUser == nil {
		t.Fatal("ExchangeCodeForUser() returned nil user")
	}
	if gotUser.ID != expectedUser.ID {
		t.Fatalf("user ID = %s, want %s", gotUser.ID, expectedUser.ID)
	}
}

func TestAuthService_ExchangeCodeForUser_InvalidCode(t *testing.T) {
	expectedErr := errors.New("invalid code")

	svc := NewAuthService(
		&mockUserRepo{},
		&mockSessionRepo{},
		&MockGitHubClient{
			ExchangeCodeFunc: func(ctx context.Context, code string) (string, error) {
				return "", expectedErr
			},
		},
		OAuthConfig{},
	)

	_, err := svc.ExchangeCodeForUser(context.Background(), "invalid")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("ExchangeCodeForUser() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_ValidateSession_Valid(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	expectedUser := &models.User{ID: userID, Role: "USER"}

	svc := NewAuthService(
		&mockUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
				if id != userID {
					t.Fatalf("GetByID() id = %s, want %s", id, userID)
				}
				return expectedUser, nil
			},
		},
		&mockSessionRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
				if id != sessionID {
					t.Fatalf("GetByID() id = %s, want %s", id, sessionID)
				}
				return &models.Session{ID: sessionID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		&MockGitHubClient{},
		OAuthConfig{},
	)

	gotUser, err := svc.ValidateSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}
	if gotUser == nil {
		t.Fatal("ValidateSession() returned nil user")
	}
	if gotUser.ID != expectedUser.ID {
		t.Fatalf("user ID = %s, want %s", gotUser.ID, expectedUser.ID)
	}
}

func TestAuthService_ValidateSession_Expired(t *testing.T) {
	sessionID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
				t.Fatal("GetByID() should not be called for expired session")
				return nil, nil
			},
		},
		&mockSessionRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
				return &models.Session{ID: sessionID, UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Minute)}, nil
			},
		},
		&MockGitHubClient{},
		OAuthConfig{},
	)

	_, err := svc.ValidateSession(context.Background(), sessionID)
	if err == nil {
		t.Fatal("ValidateSession() error = nil, want non-nil")
	}
}

func TestAuthService_ValidateSession_NotFound(t *testing.T) {
	sessionID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{},
		&mockSessionRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
				return nil, nil
			},
		},
		&MockGitHubClient{},
		OAuthConfig{},
	)

	_, err := svc.ValidateSession(context.Background(), sessionID)
	if err == nil {
		t.Fatal("ValidateSession() error = nil, want non-nil")
	}
}
