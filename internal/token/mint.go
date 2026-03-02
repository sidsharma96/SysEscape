package token

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// MintRunToken mints a signed HS256 JWT for a specific user-run-engine scope.
func MintRunToken(secret string, in MintRunTokenInput) (string, error) {
	if err := in.validate(secret); err != nil {
		return "", err
	}

	now := normalizeNow(in.Now)
	claims := RunTokenClaims{
		RunID:  in.RunID.String(),
		Engine: in.Engine,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   in.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(in.TTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("%w: sign token: %v", ErrInvalidToken, err)
	}
	return signed, nil
}
