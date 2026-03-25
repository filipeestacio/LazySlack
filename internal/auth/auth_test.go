package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/filipeestacio/lazyslack/internal/config"
)

func TestValidateSessionToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer xoxc-valid" {
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid_auth"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "user_id": "U1", "user": "test", "team_id": "T1", "team": "testteam",
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{Method: "session_token", Token: "xoxc-valid", Cookie: "d=xoxd-test"},
	}

	info, err := ValidateToken(cfg, srv.URL)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if info.Team != "testteam" {
		t.Errorf("Team = %q, want %q", info.Team, "testteam")
	}
}

func TestValidateInvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid_auth"})
	}))
	defer srv.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{Method: "session_token", Token: "xoxc-bad"},
	}

	_, err := ValidateToken(cfg, srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
