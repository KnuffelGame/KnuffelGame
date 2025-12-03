package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/pkg/config"
)

func TestAuthentication_Workflow1_ValidCookie(t *testing.T) {
	// Mock AuthService
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/validate" && r.Method == "POST" {
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)
			if req["token"] == "validtoken" {
				resp := map[string]interface{}{
					"valid":    true,
					"user_id":  "123",
					"username": "testuser",
					"is_guest": false,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			} else {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer authServer.Close()

	cfg := &config.Config{
		AuthServiceURL: authServer.URL,
		CookieDomain:   "localhost",
		CookieSameSite: "Lax",
		CookieSecure:   false,
	}

	// Test handler that checks context
	called := false
	var capturedUserID, capturedUsername string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if userID, ok := r.Context().Value(userIDKey).(string); ok {
			capturedUserID = userID
		}
		if username, ok := r.Context().Value(usernameKey).(string); ok {
			capturedUsername = username
		}
		w.WriteHeader(http.StatusOK)
	})

	// Chain middlewares
	handler := Authentication(cfg)(HeaderInjection(testHandler))

	// Create request with valid cookie
	req := httptest.NewRequest("GET", "/some/path", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "validtoken"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}
	if capturedUserID != "123" {
		t.Errorf("Expected userID 123, got %s", capturedUserID)
	}
	if capturedUsername != "testuser" {
		t.Errorf("Expected username testuser, got %s", capturedUsername)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthentication_Workflow1_InvalidCookie(t *testing.T) {
	// Mock AuthService
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/validate" && r.Method == "POST" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer authServer.Close()

	cfg := &config.Config{
		AuthServiceURL: authServer.URL,
	}

	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	handler := Authentication(cfg)(testHandler)

	// Request with invalid cookie
	req := httptest.NewRequest("GET", "/some/path", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "invalidtoken"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthentication_Workflow2_PostLobbies(t *testing.T) {
	// Mock AuthService
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/create" && r.Method == "POST" {
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)
			if req["username"] == "testuser" {
				resp := map[string]interface{}{
					"token":    "newtoken",
					"username": "testuser",
					"user_id":  "123",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			} else {
				http.Error(w, "Failed", http.StatusBadRequest)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer authServer.Close()

	cfg := &config.Config{
		AuthServiceURL: authServer.URL,
		CookieDomain:   "localhost",
		CookieSameSite: "Lax",
		CookieSecure:   false,
	}

	// Test handler that checks headers
	called := false
	var capturedUserID, capturedUsername string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		capturedUserID = r.Header.Get("X-User-ID")
		capturedUsername = r.Header.Get("X-Username")
		w.WriteHeader(http.StatusOK)
	})

	handler := Authentication(cfg)(HeaderInjection(testHandler))

	// POST /lobbies
	body := `{"username": "testuser"}`
	req := httptest.NewRequest("POST", "/lobbies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}
	if capturedUserID != "123" {
		t.Errorf("Expected X-User-ID 123, got %s", capturedUserID)
	}
	if capturedUsername != "testuser" {
		t.Errorf("Expected X-Username testuser, got %s", capturedUsername)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	// Check Set-Cookie
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "jwt" || cookies[0].Value != "newtoken" {
		t.Errorf("Expected Set-Cookie jwt=newtoken, got %v", cookies)
	}
}
