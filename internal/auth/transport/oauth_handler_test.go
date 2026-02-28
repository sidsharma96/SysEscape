package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/auth/service"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

type oauthTestUserRepo struct {
	upsertByGitHubIDFn func(ctx context.Context, githubID int64, username, displayName string) (*models.User, error)
	getByIDFn          func(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

func (m *oauthTestUserRepo) UpsertByGitHubID(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
	if m.upsertByGitHubIDFn == nil {
		return nil, nil
	}
	return m.upsertByGitHubIDFn(ctx, githubID, username, displayName)
}

func (m *oauthTestUserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, userID)
}

type oauthTestSessionRepo struct {
	createFn  func(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error)
	getByIDFn func(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	deleteFn  func(ctx context.Context, sessionID uuid.UUID) error
}

func (m *oauthTestSessionRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, expiresAt)
}

func (m *oauthTestSessionRepo) GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, sessionID)
}

func (m *oauthTestSessionRepo) Delete(ctx context.Context, sessionID uuid.UUID) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(ctx, sessionID)
}

func TestHandleGitHubCallback_Success(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	authService := service.NewAuthService(
		&oauthTestUserRepo{
			upsertByGitHubIDFn: func(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
				return &models.User{ID: userID, GitHubID: githubID, GitHubUsername: username, DisplayName: displayName, Role: "USER"}, nil
			},
		},
		&oauthTestSessionRepo{
			createFn: func(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
				return &models.Session{ID: sessionID, UserID: userID, ExpiresAt: expiresAt, CreatedAt: time.Now()}, nil
			},
		},
		&service.MockGitHubClient{
			ExchangeCodeFunc: func(ctx context.Context, code string) (string, error) {
				return "token", nil
			},
			GetUserFunc: func(ctx context.Context, accessToken string) (service.GitHubUser, error) {
				return service.GitHubUser{ID: 123, Login: "octocat", Name: "Octo Cat"}, nil
			},
		},
		service.OAuthConfig{},
	)

	handler := HandleGitHubCallback(authService)

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=valid", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusFound)
	}
	if got := res.Header.Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want %q", got, "/")
	}

	var sessionCookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session_id cookie to be set")
	}
	if sessionCookie.Value != sessionID.String() {
		t.Fatalf("cookie value = %q, want %q", sessionCookie.Value, sessionID.String())
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected HttpOnly session cookie")
	}
	if !sessionCookie.Secure {
		t.Fatal("expected Secure session cookie")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want %v", sessionCookie.SameSite, http.SameSiteLaxMode)
	}
}

func TestHandleGitHubCallback_MissingCode(t *testing.T) {
	handler := HandleGitHubCallback(nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
