package transport

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

var systemAdminUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// ContextWithUser attaches a user object to context using the transport user key.
func ContextWithUser(ctx context.Context, user *models.User) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userKey, user)
}

// AdminAPIKeyMiddleware injects an ADMIN user in context when bearer token matches apiKey.
func AdminAPIKeyMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header != "" {
				if token, ok := strings.CutPrefix(header, "Bearer "); ok {
					if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) == 1 {
						ctx := ContextWithUser(r.Context(), &models.User{ID: systemAdminUserID, Role: "ADMIN"})
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
