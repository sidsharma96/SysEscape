package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAPIKeyMiddleware_ValidKey_InjectsAdmin(t *testing.T) {
	h := AdminAPIKeyMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.Role != "ADMIN" {
			t.Fatalf("Role = %q, want ADMIN", user.Role)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestAdminAPIKeyMiddleware_InvalidKey_PassesThrough(t *testing.T) {
	h := AdminAPIKeyMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); ok {
			t.Fatal("expected no user in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer nope")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestAdminAPIKeyMiddleware_NoHeader_PassesThrough(t *testing.T) {
	h := AdminAPIKeyMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); ok {
			t.Fatal("expected no user in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}
