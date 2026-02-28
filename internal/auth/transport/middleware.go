package transport

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/auth/service"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

type userContextKey struct{}

var userKey = userContextKey{}

// SessionMiddleware attaches authenticated user to context when session is valid.
func SessionMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" || authService == nil {
				next.ServeHTTP(w, r)
				return
			}

			sessionID, err := uuid.Parse(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authService.ValidateSession(r.Context(), sessionID)
			if err != nil || user == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth enforces authentication for protected routes.
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := UserFromContext(r.Context()); !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin enforces ADMIN role for protected admin routes.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if user.Role != "ADMIN" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext returns authenticated user from request context.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	if ctx == nil {
		return nil, false
	}
	user, ok := ctx.Value(userKey).(*models.User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}
