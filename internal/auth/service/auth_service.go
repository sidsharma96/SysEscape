package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

const sessionTTL = 7 * 24 * time.Hour

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrUserNotFound    = errors.New("user not found")
)

// UserRepo defines operations required by the auth service for users.
type UserRepo interface {
	UpsertByGitHubID(ctx context.Context, githubID int64, username string, displayName string) (*models.User, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

// SessionRepo defines operations required by the auth service for sessions.
type SessionRepo interface {
	Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error)
	GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
}

// OAuthConfig stores GitHub OAuth credentials and callback URL.
type OAuthConfig struct {
	ClientID               string
	ClientSecret           string
	RedirectURL            string
	PostLoginRedirectURL   string
}

// AuthService orchestrates GitHub OAuth login and session management.
type AuthService struct {
	userRepo     UserRepo
	sessionRepo  SessionRepo
	githubClient GitHubClient
	config       OAuthConfig
}

// NewAuthService constructs an AuthService with all required dependencies.
func NewAuthService(userRepo UserRepo, sessionRepo SessionRepo, githubClient GitHubClient, config OAuthConfig) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		githubClient: githubClient,
		config:       config,
	}
}

// ExchangeCodeForUser exchanges an OAuth code for a GitHub user and upserts it locally.
func (s *AuthService) ExchangeCodeForUser(ctx context.Context, code string) (*models.User, error) {
	if s == nil {
		return nil, errors.New("auth service is nil")
	}
	if code == "" {
		return nil, errors.New("oauth code is required")
	}
	if s.githubClient == nil {
		return nil, errors.New("github client is nil")
	}
	if s.userRepo == nil {
		return nil, errors.New("user repo is nil")
	}

	accessToken, err := s.githubClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange github code: %w", err)
	}

	githubUser, err := s.githubClient.GetUser(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("get github user: %w", err)
	}

	displayName := githubUser.Name
	if displayName == "" {
		displayName = githubUser.Login
	}

	user, err := s.userRepo.UpsertByGitHubID(ctx, githubUser.ID, githubUser.Login, displayName)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

// CreateSession creates a new 7-day session for the user.
func (s *AuthService) CreateSession(ctx context.Context, userID uuid.UUID) (*models.Session, error) {
	if s == nil {
		return nil, errors.New("auth service is nil")
	}
	if s.sessionRepo == nil {
		return nil, errors.New("session repo is nil")
	}

	expiresAt := time.Now().Add(sessionTTL)
	session, err := s.sessionRepo.Create(ctx, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

// ValidateSession validates session existence and expiry, then loads the user.
func (s *AuthService) ValidateSession(ctx context.Context, sessionID uuid.UUID) (*models.User, error) {
	if s == nil {
		return nil, errors.New("auth service is nil")
	}
	if s.sessionRepo == nil {
		return nil, errors.New("session repo is nil")
	}
	if s.userRepo == nil {
		return nil, errors.New("user repo is nil")
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if !session.ExpiresAt.After(time.Now()) {
		return nil, ErrSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// Logout invalidates the session.
func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if s == nil {
		return errors.New("auth service is nil")
	}
	if s.sessionRepo == nil {
		return errors.New("session repo is nil")
	}
	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
