package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyRunToken verifies a signed run token and enforces expected run and engine scope.
func VerifyRunToken(in VerifyRunTokenInput) (*RunTokenClaims, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}

	claims := &RunTokenClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(func() time.Time { return normalizeNow(in.Now) }),
	)

	token, err := parser.ParseWithClaims(in.Token, claims, func(token *jwt.Token) (any, error) {
		return []byte(in.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %v", ErrExpiredToken, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if err := claims.validate(); err != nil {
		return nil, err
	}

	if claims.Engine != in.ExpectedEngine {
		return nil, fmt.Errorf("%w: token engine %q, expected %q", ErrEngineMismatch, claims.Engine, in.ExpectedEngine)
	}
	if claims.RunID != in.ExpectedRunID.String() {
		return nil, fmt.Errorf("%w: token runId %q, expected %q", ErrRunIDMismatch, claims.RunID, in.ExpectedRunID.String())
	}

	return claims, nil
}
