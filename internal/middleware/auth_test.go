package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"awb-gen/internal/middleware"
)

func assertResponseCode(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected %d, got %d", expected, actual)
	}
}

func mockHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// 2.1 TestAuth_ValidSingleKey
func TestAuth_ValidSingleKey(t *testing.T) {
	h := middleware.Auth("secret-key-1")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", "secret-key-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusOK, rr.Code)
}

// 2.2 TestAuth_MultiKeyList
func TestAuth_MultiKeyList(t *testing.T) {
	h := middleware.Auth("prod-key,staging-key")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", "staging-key")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusOK, rr.Code)
}

// 2.3 TestAuth_MissingHeader
func TestAuth_MissingHeader(t *testing.T) {
	h := middleware.Auth("secret-key")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusUnauthorized, rr.Code)
}

// 2.4 TestAuth_WrongKey
func TestAuth_WrongKey(t *testing.T) {
	h := middleware.Auth("prod-key")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusUnauthorized, rr.Code)
}

// 2.5 TestAuth_EmptyKey
func TestAuth_EmptyKey(t *testing.T) {
	h := middleware.Auth("secret-key")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusUnauthorized, rr.Code)
}

// 2.6 TestAuth_KeyWithPaddingWhitespace
func TestAuth_KeyWithPaddingWhitespace(t *testing.T) {
	h := middleware.Auth("prod-key")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", " prod-key ")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusUnauthorized, rr.Code)
}

// 2.7 TestAuth_NoKeysConfigured
func TestAuth_NoKeysConfigured(t *testing.T) {
	h := middleware.Auth("")(mockHandler())
	req := httptest.NewRequest(http.MethodPost, "/generate", nil)
	req.Header.Set("X-API-Key", "anything")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertResponseCode(t, http.StatusUnauthorized, rr.Code)
}
