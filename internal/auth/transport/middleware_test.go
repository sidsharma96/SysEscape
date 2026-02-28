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

type middlewareTestUserRepo struct {
	upsertByGitHubIDFn func(ctx context.Context, githubID int64, username, displayName string) (*models.User, error)
	getByIDFn          func(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

func (m *middlewareTestUserRepo) UpsertByGitHubID(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
	if m.upsertByGitHubIDFn == nil {
		return nil, nil
	}
	return m.upsertByGitHubIDFn(ctx, githubID, username, displayName)
}

func (m *middlewareTestUserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, userID)
}

type middlewareTestSessionRepo struct {
	createFn  func(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error)
	getByIDFn func(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	deleteFn  func(ctx context.Context, sessionID uuid.UUID) error
}

func (m *middlewareTestSessionRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, expiresAt)
}

func (m *middlewareTestSessionRepo) GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, sessionID)
}

func (m *middlewareTestSessionRepo) Delete(ctx context.Context, sessionID uuid.UUID) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(ctx, sessionID)
}

func TestSessionMiddleware_WithValidSession(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	authService := service.NewAuthService(
		&middlewareTestUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, Role: "USER"}, nil
			},
		},
		&middlewareTestSessionRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
				return &models.Session{ID: id, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		&service.MockGitHubClient{},
		service.OAuthConfig{},
	)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.ID != userID {
			t.Fatalf("user ID = %s, want %s", user.ID, userID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := SessionMiddleware(authService)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestSessionMiddleware_WithoutCookie(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if _, ok := UserFromContext(r.Context()); ok {
			t.Fatal("expected no user in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := SessionMiddleware(nil)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestRequireAuth_Unauthenticated(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := RequireAuth()(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequireAdmin_NonAdmin(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	authService := service.NewAuthService(
		&middlewareTestUserRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, Role: "USER"}, nil
			},
		},
		&middlewareTestSessionRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
				return &models.Session{ID: id, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		&service.MockGitHubClient{},
		service.OAuthConfig{},
	)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := SessionMiddleware(authService)(RequireAdmin()(next))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}
