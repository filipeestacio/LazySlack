package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"user_id": "U123",
			"user":    "testuser",
			"team_id": "T123",
			"team":    "testteam",
			"url":     "https://testteam.slack.com/",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "xoxc-test", "")
	info, err := c.AuthTest()
	if err != nil {
		t.Fatalf("AuthTest() error: %v", err)
	}
	if info.UserID != "U123" {
		t.Errorf("UserID = %q, want %q", info.UserID, "U123")
	}
	if info.Team != "testteam" {
		t.Errorf("Team = %q, want %q", info.Team, "testteam")
	}
}

func TestListChannels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{"id": "C1", "name": "general", "is_member": true, "is_private": false},
				{"id": "C2", "name": "random", "is_member": true, "is_private": false},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "xoxc-test", "")
	channels, err := c.ListChannels()
	if err != nil {
		t.Fatalf("ListChannels() error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}
	if channels[0].Name != "general" {
		t.Errorf("channels[0].Name = %q, want %q", channels[0].Name, "general")
	}
}

func TestSendMessage(t *testing.T) {
	var gotChannel, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotChannel = r.FormValue("channel")
		gotText = r.FormValue("text")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "xoxc-test", "")
	err := c.SendMessage("C1", "hello world")
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}
	if gotChannel != "C1" {
		t.Errorf("channel = %q, want %q", gotChannel, "C1")
	}
	if gotText != "hello world" {
		t.Errorf("text = %q, want %q", gotText, "hello world")
	}
}

func newTestClient(baseURL, token, cookie string) *Client {
	return NewClient(token, cookie, WithBaseURL(baseURL))
}
