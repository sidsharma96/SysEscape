package transport

import (
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/auth/service"
)

const sessionCookieName = "session_id"

// HandleGitHubLogin redirects the user to GitHub OAuth authorize endpoint.
func HandleGitHubLogin(cfg service.OAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := url.Values{}
		q.Set("client_id", cfg.ClientID)
		q.Set("redirect_uri", cfg.RedirectURL)

		redirectURL := "https://github.com/login/oauth/authorize?" + q.Encode()
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// HandleGitHubCallback handles OAuth callback, creates a session, and sets cookie.
func HandleGitHubCallback(authService *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing oauth code", http.StatusBadRequest)
			return
		}
		if authService == nil {
			http.Error(w, "auth service unavailable", http.StatusInternalServerError)
			return
		}

		user, err := authService.ExchangeCodeForUser(r.Context(), code)
		if err != nil {
			http.Error(w, "oauth exchange failed", http.StatusUnauthorized)
			return
		}

		session, err := authService.CreateSession(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "session creation failed", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    session.ID.String(),
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  session.ExpiresAt,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// HandleLogout clears session from store and cookie, then redirects to root.
func HandleLogout(authService *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if sessionID, parseErr := uuid.Parse(cookie.Value); parseErr == nil && authService != nil {
				_ = authService.Logout(r.Context(), sessionID)
			}
		}

		clearSessionCookie(w)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
