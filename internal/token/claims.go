package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	EngineA = "A"
	EngineB = "B"
)

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("expired token")
	ErrEngineMismatch = errors.New("engine mismatch")
	ErrRunIDMismatch  = errors.New("runId mismatch")
	ErrInvalidClaims  = errors.New("invalid claims")
)

// RunTokenClaims defines the JWT claims used by runToken.
type RunTokenClaims struct {
	RunID  string `json:"runId"`
	Engine string `json:"engine"`
	jwt.RegisteredClaims
}

func (c *RunTokenClaims) validate() error {
	if strings.TrimSpace(c.Subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidClaims)
	}
	if strings.TrimSpace(c.RunID) == "" {
		return fmt.Errorf("%w: runId is required", ErrInvalidClaims)
	}
	if err := validateEngine(c.Engine); err != nil {
		return err
	}
	if c.IssuedAt == nil {
		return fmt.Errorf("%w: iat is required", ErrInvalidClaims)
	}
	if c.ExpiresAt == nil {
		return fmt.Errorf("%w: exp is required", ErrInvalidClaims)
	}
	if c.ExpiresAt.Time.Before(c.IssuedAt.Time) {
		return fmt.Errorf("%w: exp must be after iat", ErrInvalidClaims)
	}
	return nil
}

// MintRunTokenInput contains all required fields for minting a run token.
type MintRunTokenInput struct {
	UserID uuid.UUID
	RunID  uuid.UUID
	Engine string
	TTL    time.Duration
	Now    time.Time
}

func (in MintRunTokenInput) validate(secret string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("%w: secret is required", ErrInvalidClaims)
	}
	if in.UserID == uuid.Nil {
		return fmt.Errorf("%w: userId is required", ErrInvalidClaims)
	}
	if in.RunID == uuid.Nil {
		return fmt.Errorf("%w: runId is required", ErrInvalidClaims)
	}
	if err := validateEngine(in.Engine); err != nil {
		return err
	}
	if in.TTL <= 0 {
		return fmt.Errorf("%w: ttl must be positive", ErrInvalidClaims)
	}
	return nil
}

// VerifyRunTokenInput contains all required fields for verifying a run token.
type VerifyRunTokenInput struct {
	Token          string
	Secret         string
	ExpectedRunID  uuid.UUID
	ExpectedEngine string
	Now            time.Time
}

func (in VerifyRunTokenInput) validate() error {
	if strings.TrimSpace(in.Token) == "" {
		return fmt.Errorf("%w: token is required", ErrInvalidClaims)
	}
	if strings.TrimSpace(in.Secret) == "" {
		return fmt.Errorf("%w: secret is required", ErrInvalidClaims)
	}
	if in.ExpectedRunID == uuid.Nil {
		return fmt.Errorf("%w: expected runId is required", ErrInvalidClaims)
	}
	if err := validateEngine(in.ExpectedEngine); err != nil {
		return err
	}
	return nil
}

func validateEngine(engine string) error {
	switch engine {
	case EngineA, EngineB:
		return nil
	default:
		return fmt.Errorf("%w: engine must be one of %q or %q", ErrInvalidClaims, EngineA, EngineB)
	}
}

func normalizeNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
