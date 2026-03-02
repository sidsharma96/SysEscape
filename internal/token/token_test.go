package token

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestMintAndVerifyRunToken(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	uid, rid := uuid.New(), uuid.New()

	tok, err := MintRunToken("secret", MintRunTokenInput{UserID: uid, RunID: rid, Engine: EngineA, TTL: 30 * time.Minute, Now: now})
	if err != nil {
		t.Fatalf("MintRunToken() error = %v", err)
	}

	claims, err := VerifyRunToken(VerifyRunTokenInput{Token: tok, Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now.Add(10 * time.Minute)})
	if err != nil {
		t.Fatalf("VerifyRunToken() error = %v", err)
	}
	if claims.Subject != uid.String() || claims.RunID != rid.String() || claims.Engine != EngineA {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(now) {
		t.Fatalf("IssuedAt = %v, want %v", claims.IssuedAt, now)
	}
	wantExp := now.Add(30 * time.Minute)
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(wantExp) {
		t.Fatalf("ExpiresAt = %v, want %v", claims.ExpiresAt, wantExp)
	}
}

func TestVerifyRunTokenFailures(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	uid, rid := uuid.New(), uuid.New()

	valid := mustMint(t, "secret", MintRunTokenInput{UserID: uid, RunID: rid, Engine: EngineA, TTL: 15 * time.Minute, Now: now})
	expired := mustMint(t, "secret", MintRunTokenInput{UserID: uid, RunID: rid, Engine: EngineA, TTL: time.Minute, Now: now})
	wrongAlg := mustMintHS512(t, "secret", RunTokenClaims{RunID: rid.String(), Engine: EngineA, RegisteredClaims: jwt.RegisteredClaims{Subject: uid.String(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute))}})

	tests := []struct {
		name string
		in   VerifyRunTokenInput
		is   error
	}{
		{"expired", VerifyRunTokenInput{Token: expired, Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now.Add(2 * time.Minute)}, ErrExpiredToken},
		{"wrong engine", VerifyRunTokenInput{Token: valid, Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineB, Now: now.Add(2 * time.Minute)}, ErrEngineMismatch},
		{"wrong runId", VerifyRunTokenInput{Token: valid, Secret: "secret", ExpectedRunID: uuid.New(), ExpectedEngine: EngineA, Now: now.Add(2 * time.Minute)}, ErrRunIDMismatch},
		{"malformed", VerifyRunTokenInput{Token: "not-a-jwt", Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now}, ErrInvalidToken},
		{"non hs256", VerifyRunTokenInput{Token: wrongAlg, Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now}, ErrInvalidToken},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := VerifyRunToken(tt.in)
			if !errors.Is(err, tt.is) {
				t.Fatalf("VerifyRunToken() error = %v, want %v", err, tt.is)
			}
		})
	}
}

func TestMintRunTokenInputValidation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	uid, rid := uuid.New(), uuid.New()

	tests := []MintRunTokenInput{
		{RunID: rid, Engine: EngineA, TTL: time.Minute, Now: now},
		{UserID: uid, Engine: EngineA, TTL: time.Minute, Now: now},
		{UserID: uid, RunID: rid, Engine: "C", TTL: time.Minute, Now: now},
		{UserID: uid, RunID: rid, Engine: EngineA, TTL: 0, Now: now},
	}
	for _, in := range tests {
		_, err := MintRunToken("secret", in)
		if !errors.Is(err, ErrInvalidClaims) {
			t.Fatalf("MintRunToken() error = %v, want %v", err, ErrInvalidClaims)
		}
	}
	_, err := MintRunToken("", MintRunTokenInput{UserID: uid, RunID: rid, Engine: EngineA, TTL: time.Minute, Now: now})
	if !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("MintRunToken() empty secret error = %v, want %v", err, ErrInvalidClaims)
	}
}

func TestVerifyRunTokenInputValidation(t *testing.T) {
	t.Parallel()
	now, rid := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), uuid.New()

	tests := []VerifyRunTokenInput{
		{Secret: "secret", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now},
		{Token: "abc", ExpectedRunID: rid, ExpectedEngine: EngineA, Now: now},
		{Token: "abc", Secret: "secret", ExpectedEngine: EngineA, Now: now},
		{Token: "abc", Secret: "secret", ExpectedRunID: rid, Now: now},
	}
	for _, in := range tests {
		_, err := VerifyRunToken(in)
		if !errors.Is(err, ErrInvalidClaims) {
			t.Fatalf("VerifyRunToken() error = %v, want %v", err, ErrInvalidClaims)
		}
	}
}

func mustMint(t *testing.T, secret string, in MintRunTokenInput) string {
	t.Helper()
	tok, err := MintRunToken(secret, in)
	if err != nil {
		t.Fatalf("MintRunToken() error = %v", err)
	}
	return tok
}

func mustMintHS512(t *testing.T, secret string, claims RunTokenClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
}
