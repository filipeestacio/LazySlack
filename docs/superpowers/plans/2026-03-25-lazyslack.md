# LazySlack Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a keyboard-driven terminal UI for Slack with session token and OAuth authentication, channel browsing, messaging, threading, and emoji reactions.

**Architecture:** Bubble Tea Elm architecture with a root model routing messages between self-contained panel models (sidebar, messages, thread overlay, compose overlay). A unified Slack client interface abstracts over both auth methods. Config via XDG + YAML.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, slack-go/slack, YAML, Nix

**Spec:** `docs/superpowers/specs/2026-03-25-lazyslack-design.md`

---

## File Structure

```
lazyslack/
├── flake.nix                          # Nix dev shell + build derivation
├── main.go                            # Entry point: load config, create client, launch TUI
├── internal/
│   ├── config/
│   │   └── config.go                  # XDG paths, YAML read/write, Config struct
│   ├── auth/
│   │   └── auth.go                    # Session token validation, OAuth flow (local HTTP server)
│   ├── slack/
│   │   ├── client.go                  # SlackClient interface + implementation wrapping slack-go
│   │   ├── types.go                   # Domain types: Channel, Message, User, Conversation, AuthInfo
│   │   ├── poller.go                  # Background polling goroutine, expbackoff, NewMessagesMsg
│   │   └── renderer.go               # Slack markup → Lip Gloss styled string conversion
│   ├── app/
│   │   ├── app.go                     # Root Bubble Tea model, focus routing, panel composition
│   │   └── keys.go                    # Keymap definitions (normal mode bindings)
│   ├── ui/
│   │   ├── styles/
│   │   │   └── styles.go             # Lip Gloss style constants (colors, borders, dimensions)
│   │   ├── sidebar/
│   │   │   └── sidebar.go            # Channel/DM list model with sections, filtering
│   │   ├── messages/
│   │   │   └── messages.go           # Message feed model with scrolling, pagination
│   │   ├── thread/
│   │   │   └── thread.go             # Thread overlay model (parent + replies)
│   │   ├── input/
│   │   │   └── input.go              # Compose overlay model (text area, send/cancel)
│   │   ├── emoji/
│   │   │   └── emoji.go              # Emoji picker model (fuzzy search over Unicode emoji)
│   │   ├── help/
│   │   │   └── help.go               # Help overlay model (keybinding reference)
│   │   └── statusbar/
│   │       └── statusbar.go          # Status bar: workspace, connection, mode, keybind hints
```

---

### Task 1: Nix Dev Environment + Go Module Init

**Files:**
- Create: `flake.nix`
- Create: `.envrc` (direnv integration)
- Create: `go.mod` (via `go mod init`)
- Create: `main.go` (minimal hello world)

- [ ] **Step 1: Create flake.nix**

```nix
{
  description = "LazySlack - Terminal UI for Slack";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "lazyslack";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
        };
      });
}
```

- [ ] **Step 2: Create .envrc for direnv**

```
use flake
```

- [ ] **Step 3: Enter nix shell and init Go module**

Run: `nix develop --command bash -c "go mod init github.com/filipeestacio/lazyslack"`
Expected: `go.mod` created

- [ ] **Step 4: Create minimal main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("lazyslack")
}
```

- [ ] **Step 5: Verify it builds and runs**

Run: `nix develop --command bash -c "go run ."`
Expected: prints `lazyslack`

- [ ] **Step 6: Add .envrc and .superpowers to .gitignore**

Append `!.envrc` is not needed — `.envrc` should be committed. Ensure `.gitignore` has:
```
.superpowers/
```

- [ ] **Step 7: Commit**

```bash
git add flake.nix .envrc go.mod main.go .gitignore
git commit -m "feat: init nix dev environment and go module"
```

---

### Task 2: Config Layer

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for config**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := configDir()
	want := filepath.Join(home, ".config", "lazyslack")
	if got != want {
		t.Errorf("configDir() = %q, want %q", got, want)
	}
}

func TestCustomXDGConfigPath(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)

	got := configDir()
	want := filepath.Join(custom, "lazyslack")
	if got != want {
		t.Errorf("configDir() = %q, want %q", got, want)
	}
}

func TestLoadCreatesDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Auth.Method != "" {
		t.Errorf("expected empty auth method for new config, got %q", cfg.Auth.Method)
	}

	configFile := filepath.Join(home, ".config", "lazyslack", "config.yaml")
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSaveAndLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg := &Config{
		Auth: AuthConfig{
			Method: "session_token",
			Token:  "xoxc-test",
			Cookie: "d=xoxd-test",
		},
		Workspace: WorkspaceConfig{Name: "testcorp"},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Auth.Token != "xoxc-test" {
		t.Errorf("token = %q, want %q", loaded.Auth.Token, "xoxc-test")
	}
	if loaded.Workspace.Name != "testcorp" {
		t.Errorf("workspace = %q, want %q", loaded.Workspace.Name, "testcorp")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: compilation error — package doesn't exist yet

- [ ] **Step 3: Add gopkg.in/yaml.v3 dependency**

Run: `go get gopkg.in/yaml.v3`

- [ ] **Step 4: Implement config.go**

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Auth      AuthConfig      `yaml:"auth"`
	Workspace WorkspaceConfig `yaml:"workspace"`
}

type AuthConfig struct {
	Method            string `yaml:"method"`
	Token             string `yaml:"token"`
	Cookie            string `yaml:"cookie,omitempty"`
	OAuthClientID     string `yaml:"oauth_client_id,omitempty"`
	OAuthClientSecret string `yaml:"oauth_client_secret,omitempty"`
}

type WorkspaceConfig struct {
	Name string `yaml:"name"`
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lazyslack")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazyslack")
}

func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func Load() (*Config, error) {
	path := configPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := &Config{}
		if err := Save(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0600)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: all 4 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config layer with XDG paths and YAML persistence"
```

---

### Task 3: Slack Domain Types

**Files:**
- Create: `internal/slack/types.go`

- [ ] **Step 1: Define domain types**

```go
package slack

import "time"

type AuthInfo struct {
	UserID   string
	User     string
	TeamID   string
	Team     string
	URL      string
}

type Channel struct {
	ID        string
	Name      string
	IsPrivate bool
	IsMember  bool
	Topic     string
}

type Conversation struct {
	ID     string
	UserID string
	Name   string
}

type User struct {
	ID          string
	Name        string
	DisplayName string
	IsBot       bool
}

type Message struct {
	UserID    string
	Username  string
	Text      string
	Timestamp string
	ThreadTS  string
	ReplyCount int
	Reactions []Reaction
	Edited    bool
}

type Reaction struct {
	Name  string
	Count int
	Users []string
}

type HistoryResult struct {
	Messages []Message
	Cursor   string
	HasMore  bool
}

func (m Message) Time() time.Time {
	// Slack timestamps are Unix epoch with microseconds as decimal
	// e.g., "1706000000.000000"
	var sec, usec int64
	for i, c := range m.Timestamp {
		if c == '.' {
			sec = parseInt(m.Timestamp[:i])
			usec = parseInt(m.Timestamp[i+1:])
			break
		}
	}
	return time.Unix(sec, usec*1000)
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/slack/`
Expected: success, no output

- [ ] **Step 3: Commit**

```bash
git add internal/slack/types.go
git commit -m "feat: add slack domain types"
```

---

### Task 4: Slack Client Interface + Implementation

**Files:**
- Create: `internal/slack/client.go`
- Create: `internal/slack/client_test.go`

- [ ] **Step 1: Write failing tests for client**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/slack/ -v -run "TestAuth|TestList|TestSend"`
Expected: compilation error — Client doesn't exist

- [ ] **Step 3: Add slack-go dependency**

Run: `go get github.com/slack-go/slack`

- [ ] **Step 4: Implement client.go**

```go
package slack

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	slackapi "github.com/slack-go/slack"
)

type SlackClient interface {
	AuthTest() (*AuthInfo, error)
	ListChannels() ([]Channel, error)
	ListDMs() ([]Conversation, error)
	GetHistory(channelID string, cursor string) (*HistoryResult, error)
	GetThreadReplies(channelID, threadTS string) ([]Message, error)
	SendMessage(channelID, text string) error
	ReplyToThread(channelID, threadTS, text string) error
	AddReaction(channelID, timestamp, emoji string) error
	RemoveReaction(channelID, timestamp, emoji string) error
	GetUsers() ([]User, error)
}

type Client struct {
	api     *slackapi.Client
	baseURL string
	token   string
	cookie  string
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

func NewClient(token, cookie string, opts ...ClientOption) *Client {
	c := &Client{token: token, cookie: cookie}
	for _, o := range opts {
		o(c)
	}

	options := []slackapi.Option{}
	if cookie != "" {
		options = append(options, slackapi.OptionHTTPClient(&http.Client{
			Transport: &cookieTransport{cookie: cookie},
		}))
	}
	if c.baseURL != "" {
		options = append(options, slackapi.OptionAPIURL(c.baseURL+"/"))
	}

	c.api = slackapi.New(token, options...)
	return c
}

type cookieTransport struct {
	cookie string
}

func (t *cookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", t.cookie)
	return http.DefaultTransport.RoundTrip(req)
}

func (c *Client) AuthTest() (*AuthInfo, error) {
	resp, err := c.api.AuthTest()
	if err != nil {
		return nil, err
	}
	return &AuthInfo{
		UserID: resp.UserID,
		User:   resp.User,
		TeamID: resp.TeamID,
		Team:   resp.Team,
		URL:    resp.URL,
	}, nil
}

func (c *Client) ListChannels() ([]Channel, error) {
	params := &slackapi.GetConversationsParameters{
		Types:           []string{"public_channel", "private_channel"},
		ExcludeArchived: true,
		Limit:           1000,
	}

	var channels []Channel
	for {
		convs, cursor, err := c.api.GetConversations(params)
		if err != nil {
			return nil, err
		}
		for _, ch := range convs {
			channels = append(channels, Channel{
				ID:        ch.ID,
				Name:      ch.Name,
				IsPrivate: ch.IsPrivate,
				IsMember:  ch.IsMember,
				Topic:     ch.Topic.Value,
			})
		}
		if cursor == "" {
			break
		}
		params.Cursor = cursor
	}
	return channels, nil
}

func (c *Client) ListDMs() ([]Conversation, error) {
	params := &slackapi.GetConversationsParameters{
		Types: []string{"im", "mpim"},
		Limit: 200,
	}

	var convs []Conversation
	chans, _, err := c.api.GetConversations(params)
	if err != nil {
		return nil, err
	}
	for _, ch := range chans {
		convs = append(convs, Conversation{
			ID:     ch.ID,
			UserID: ch.User,
			Name:   ch.User,
		})
	}
	return convs, nil
}

func (c *Client) GetHistory(channelID string, cursor string) (*HistoryResult, error) {
	params := &slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     50,
		Cursor:    cursor,
	}

	resp, err := c.api.GetConversationHistory(params)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		messages = append(messages, convertMessage(m))
	}

	return &HistoryResult{
		Messages: messages,
		Cursor:   resp.ResponseMetaData.NextCursor,
		HasMore:  resp.HasMore,
	}, nil
}

func (c *Client) GetThreadReplies(channelID, threadTS string) ([]Message, error) {
	msgs, _, _, err := c.api.GetConversationReplies(&slackapi.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	})
	if err != nil {
		return nil, err
	}

	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, convertMessage(m))
	}
	return result, nil
}

func (c *Client) SendMessage(channelID, text string) error {
	if c.baseURL != "" {
		return c.sendMessageHTTP(channelID, text, "")
	}
	_, _, err := c.api.PostMessage(channelID, slackapi.MsgOptionText(text, false))
	return err
}

func (c *Client) ReplyToThread(channelID, threadTS, text string) error {
	if c.baseURL != "" {
		return c.sendMessageHTTP(channelID, text, threadTS)
	}
	_, _, err := c.api.PostMessage(channelID,
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionTS(threadTS),
	)
	return err
}

func (c *Client) sendMessageHTTP(channelID, text, threadTS string) error {
	data := url.Values{
		"channel": {channelID},
		"text":    {text},
	}
	if threadTS != "" {
		data.Set("thread_ts", threadTS)
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://slack.com/api"
	}

	req, err := http.NewRequest("POST", baseURL+"/chat.postMessage",
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) AddReaction(channelID, timestamp, emoji string) error {
	return c.api.AddReaction(emoji, slackapi.ItemRef{
		Channel:   channelID,
		Timestamp: timestamp,
	})
}

func (c *Client) RemoveReaction(channelID, timestamp, emoji string) error {
	return c.api.RemoveReaction(emoji, slackapi.ItemRef{
		Channel:   channelID,
		Timestamp: timestamp,
	})
}

func (c *Client) GetUsers() ([]User, error) {
	slackUsers, err := c.api.GetUsers()
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(slackUsers))
	for _, u := range slackUsers {
		displayName := u.Profile.DisplayName
		if displayName == "" {
			displayName = u.RealName
		}
		users = append(users, User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: displayName,
			IsBot:       u.IsBot,
		})
	}
	return users, nil
}

func convertMessage(m slackapi.Message) Message {
	var reactions []Reaction
	for _, r := range m.Reactions {
		reactions = append(reactions, Reaction{
			Name:  r.Name,
			Count: r.Count,
			Users: r.Users,
		})
	}

	return Message{
		UserID:     m.User,
		Username:   m.Username,
		Text:       m.Text,
		Timestamp:  m.Timestamp,
		ThreadTS:   m.ThreadTimestamp,
		ReplyCount: m.ReplyCount,
		Reactions:  reactions,
		Edited:     m.Edited != nil,
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/slack/ -v -run "TestAuth|TestList|TestSend"`
Expected: all 3 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/slack/client.go internal/slack/client_test.go go.mod go.sum
git commit -m "feat: add slack client interface and implementation"
```

---

### Task 5: Message Renderer

**Files:**
- Create: `internal/slack/renderer.go`
- Create: `internal/slack/renderer_test.go`

- [ ] **Step 1: Write failing tests for renderer**

```go
package slack

import "testing"

type mockUserResolver struct {
	users    map[string]string
	channels map[string]string
}

func (m *mockUserResolver) ResolveUser(id string) string {
	if name, ok := m.users[id]; ok {
		return name
	}
	return id
}

func (m *mockUserResolver) ResolveChannel(id string) string {
	if name, ok := m.channels[id]; ok {
		return name
	}
	return id
}

func TestRenderBold(t *testing.T) {
	r := NewRenderer(nil)
	got := r.RenderPlain("hello *world*")
	want := "hello world"
	if got != want {
		t.Errorf("RenderPlain(*bold*) = %q, want %q", got, want)
	}
}

func TestRenderUserMention(t *testing.T) {
	resolver := &mockUserResolver{
		users: map[string]string{"U123": "alice"},
	}
	r := NewRenderer(resolver)
	got := r.RenderPlain("hello <@U123>")
	want := "hello @alice"
	if got != want {
		t.Errorf("RenderPlain(mention) = %q, want %q", got, want)
	}
}

func TestRenderChannelLink(t *testing.T) {
	resolver := &mockUserResolver{
		channels: map[string]string{"C456": "general"},
	}
	r := NewRenderer(resolver)
	got := r.RenderPlain("see <#C456>")
	want := "see #general"
	if got != want {
		t.Errorf("RenderPlain(channel) = %q, want %q", got, want)
	}
}

func TestRenderURL(t *testing.T) {
	r := NewRenderer(nil)
	got := r.RenderPlain("check <https://example.com|example>")
	want := "check example (https://example.com)"
	if got != want {
		t.Errorf("RenderPlain(url) = %q, want %q", got, want)
	}
}

func TestRenderCodeInline(t *testing.T) {
	r := NewRenderer(nil)
	got := r.RenderPlain("use `go test` to run")
	want := "use go test to run"
	if got != want {
		t.Errorf("RenderPlain(code) = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/slack/ -v -run "TestRender"`
Expected: compilation error — Renderer doesn't exist

- [ ] **Step 3: Implement renderer.go**

```go
package slack

import (
	"regexp"
	"strings"
)

type UserResolver interface {
	ResolveUser(id string) string
	ResolveChannel(id string) string
}

type Renderer struct {
	resolver UserResolver
}

func NewRenderer(resolver UserResolver) *Renderer {
	return &Renderer{resolver: resolver}
}

var (
	userMentionRe   = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)
	channelLinkRe   = regexp.MustCompile(`<#(C[A-Z0-9]+)(?:\|([^>]+))?>`)
	urlRe           = regexp.MustCompile(`<(https?://[^|>]+)\|([^>]+)>`)
	urlBareRe       = regexp.MustCompile(`<(https?://[^>]+)>`)
	boldRe          = regexp.MustCompile(`\*([^*]+)\*`)
	italicRe        = regexp.MustCompile(`_([^_]+)_`)
	strikeRe        = regexp.MustCompile(`~([^~]+)~`)
	codeInlineRe    = regexp.MustCompile("`([^`]+)`")
)

func (r *Renderer) RenderPlain(text string) string {
	text = userMentionRe.ReplaceAllStringFunc(text, func(match string) string {
		id := userMentionRe.FindStringSubmatch(match)[1]
		if r.resolver != nil {
			return "@" + r.resolver.ResolveUser(id)
		}
		return "@" + id
	})

	text = channelLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := channelLinkRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return "#" + parts[2]
		}
		if r.resolver != nil {
			return "#" + r.resolver.ResolveChannel(parts[1])
		}
		return "#" + parts[1]
	})

	text = urlRe.ReplaceAllString(text, "$2 ($1)")
	text = urlBareRe.ReplaceAllString(text, "$1")

	text = boldRe.ReplaceAllString(text, "$1")
	text = italicRe.ReplaceAllString(text, "$1")
	text = strikeRe.ReplaceAllString(text, "$1")
	text = codeInlineRe.ReplaceAllString(text, "$1")

	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")

	return text
}
```

Note: `RenderPlain` strips markup for plain text. A `RenderStyled` method that returns Lip Gloss styled output will be added in the UI task when styles are available.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/slack/ -v -run "TestRender"`
Expected: all 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/slack/renderer.go internal/slack/renderer_test.go
git commit -m "feat: add slack message renderer with markup conversion"
```

---

### Task 6: Polling Loop

**Files:**
- Create: `internal/slack/poller.go`
- Create: `internal/slack/poller_test.go`

- [ ] **Step 1: Write failing tests for poller**

```go
package slack

import (
	"sync"
	"testing"
	"time"
)

type mockSlackClient struct {
	mu       sync.Mutex
	history  []Message
	calls    int
	err      error
}

func (m *mockSlackClient) GetHistory(channelID, cursor string) (*HistoryResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &HistoryResult{Messages: m.history}, nil
}

func (m *mockSlackClient) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestPollerDeliversNewMessages(t *testing.T) {
	mock := &mockSlackClient{
		history: []Message{{Text: "hello", Timestamp: "1706000001.000000"}},
	}

	msgs := make(chan []Message, 10)
	p := NewPoller(mock, 50*time.Millisecond, func(m []Message) { msgs <- m })

	p.SetChannel("C1")
	p.Start()
	defer p.Stop()

	select {
	case got := <-msgs:
		if len(got) == 0 || got[0].Text != "hello" {
			t.Errorf("unexpected message: %v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for messages")
	}
}

func TestPollerPausesWithNoChannel(t *testing.T) {
	mock := &mockSlackClient{
		history: []Message{{Text: "hello", Timestamp: "1706000001.000000"}},
	}

	p := NewPoller(mock, 50*time.Millisecond, func(m []Message) {})
	p.Start()
	defer p.Stop()

	time.Sleep(150 * time.Millisecond)
	if mock.getCalls() != 0 {
		t.Errorf("expected 0 calls with no channel, got %d", mock.getCalls())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/slack/ -v -run "TestPoller"`
Expected: compilation error — Poller doesn't exist

- [ ] **Step 3: Implement poller.go**

```go
package slack

import (
	"sync"
	"time"
)

type HistoryFetcher interface {
	GetHistory(channelID, cursor string) (*HistoryResult, error)
}

type Poller struct {
	client   HistoryFetcher
	interval time.Duration
	onNew    func([]Message)

	mu        sync.Mutex
	channelID string
	lastTS    string
	stop      chan struct{}
}

func NewPoller(client HistoryFetcher, interval time.Duration, onNew func([]Message)) *Poller {
	return &Poller{
		client:   client,
		interval: interval,
		onNew:    onNew,
		stop:     make(chan struct{}),
	}
}

func (p *Poller) SetChannel(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channelID = id
	p.lastTS = ""
}

func (p *Poller) Start() {
	go p.loop()
}

func (p *Poller) Stop() {
	close(p.stop)
}

func (p *Poller) FetchNow() {
	p.poll()
}

func (p *Poller) loop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

func (p *Poller) poll() {
	p.mu.Lock()
	chID := p.channelID
	lastTS := p.lastTS
	p.mu.Unlock()

	if chID == "" {
		return
	}

	result, err := p.client.GetHistory(chID, "")
	if err != nil {
		return
	}

	if len(result.Messages) == 0 {
		return
	}

	var newMsgs []Message
	for _, m := range result.Messages {
		if m.Timestamp > lastTS {
			newMsgs = append(newMsgs, m)
		}
	}

	if len(newMsgs) > 0 {
		p.mu.Lock()
		p.lastTS = result.Messages[0].Timestamp
		p.mu.Unlock()
		p.onNew(newMsgs)
	} else if lastTS == "" {
		p.mu.Lock()
		p.lastTS = result.Messages[0].Timestamp
		p.mu.Unlock()
		p.onNew(result.Messages)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/slack/ -v -run "TestPoller"`
Expected: all 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/slack/poller.go internal/slack/poller_test.go
git commit -m "feat: add polling loop with channel switching and backoff"
```

---

### Task 7: Auth Layer

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/auth/ -v`
Expected: compilation error

- [ ] **Step 3: Implement auth.go**

```go
package auth

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/filipeestacio/lazyslack/internal/config"
	slackclient "github.com/filipeestacio/lazyslack/internal/slack"
)

func ValidateToken(cfg *config.Config, baseURL string) (*slackclient.AuthInfo, error) {
	opts := []slackclient.ClientOption{}
	if baseURL != "" {
		opts = append(opts, slackclient.WithBaseURL(baseURL))
	}

	client := slackclient.NewClient(cfg.Auth.Token, cfg.Auth.Cookie, opts...)
	return client.AuthTest()
}

func RunOAuthFlow(clientID, clientSecret string, scopes []string) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprint(w, "Error: no authorization code received. You can close this tab.")
			return
		}

		token, err := exchangeCode(clientID, clientSecret, code, redirectURI)
		if err != nil {
			errCh <- err
			fmt.Fprint(w, "Error exchanging code. Check terminal for details.")
			return
		}
		tokenCh <- token
		fmt.Fprint(w, "Authentication successful! You can close this tab and return to LazySlack.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += ","
		}
		scopeStr += s
	}

	authURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&user_scope=%s&redirect_uri=%s",
		clientID, scopeStr, redirectURI,
	)
	openBrowser(authURL)

	select {
	case token := <-tokenCh:
		return token, nil
	case err := <-errCh:
		return "", err
	}
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (string, error) {
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", map[string][]string{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AuthedUser  struct {
			AccessToken string `json:"access_token"`
		} `json:"authed_user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("oauth exchange failed: %s", result.Error)
	}
	return result.AuthedUser.AccessToken, nil
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth/ -v`
Expected: all 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/ go.mod go.sum
git commit -m "feat: add auth layer with session token validation and oauth flow"
```

---

### Task 8: Lip Gloss Styles

**Files:**
- Create: `internal/ui/styles/styles.go`

- [ ] **Step 1: Define style constants**

```go
package styles

import "github.com/charmbracelet/lipgloss"

var (
	AppBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	SidebarStyle = lipgloss.NewStyle().
			Width(30).
			Padding(1, 1)

	SidebarSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	SidebarNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	SidebarSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true).
			MarginTop(1).
			Padding(0, 1)

	MessagesStyle = lipgloss.NewStyle().
			Padding(1, 1)

	MessageUsername = lipgloss.NewStyle().
			Bold(true)

	MessageTimestamp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	MessageText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	MessageReaction = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	ThreadBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 1)

	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	StatusBarKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	UnreadDot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
)

var UsernameColors = []lipgloss.Color{
	"204", "166", "178", "114", "81", "147", "212", "209",
}

func ColorForUser(userID string) lipgloss.Color {
	var hash int
	for _, c := range userID {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return UsernameColors[hash%len(UsernameColors)]
}
```

- [ ] **Step 2: Add lip gloss dependency**

Run: `go get github.com/charmbracelet/lipgloss`

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/ui/styles/`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/ui/styles/ go.mod go.sum
git commit -m "feat: add lip gloss style definitions"
```

---

### Task 9: Status Bar Component

**Files:**
- Create: `internal/ui/statusbar/statusbar.go`
- Create: `internal/ui/statusbar/statusbar_test.go`

- [ ] **Step 1: Write failing test**

```go
package statusbar

import "testing"

func TestViewShowsWorkspace(t *testing.T) {
	m := New("testcorp")
	view := m.View(80)
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
}

func TestSetConnected(t *testing.T) {
	m := New("testcorp")
	m.SetConnected(false)
	if m.connected {
		t.Error("expected disconnected")
	}
	m.SetConnected(true)
	if !m.connected {
		t.Error("expected connected")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/statusbar/ -v`
Expected: compilation error

- [ ] **Step 3: Implement statusbar.go**

```go
package statusbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type Model struct {
	workspace   string
	connected   bool
	rateLimited bool
	hints       string
}

func New(workspace string) Model {
	return Model{
		workspace: workspace,
		connected: true,
		hints:     "c:compose  t:thread  r:react  ?:help",
	}
}

func (m *Model) SetConnected(v bool)   { m.connected = v }
func (m *Model) SetRateLimited(v bool) { m.rateLimited = v }
func (m *Model) SetHints(h string)     { m.hints = h }

func (m Model) View(width int) string {
	status := "●"
	statusColor := lipgloss.Color("114")
	if !m.connected {
		status = "○ disconnected"
		statusColor = lipgloss.Color("196")
	} else if m.rateLimited {
		status = "◐ rate limited"
		statusColor = lipgloss.Color("214")
	}

	left := styles.StatusBarKey.Render(" "+m.workspace+" ") +
		" " +
		lipgloss.NewStyle().Foreground(statusColor).Render(status)

	right := styles.StatusBarStyle.Render(m.hints)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Render(spaces(gap)) + right

	return styles.StatusBarStyle.Width(width).Render(bar)
}

func spaces(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/statusbar/ -v`
Expected: all 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/statusbar/
git commit -m "feat: add status bar component"
```

---

### Task 10: Sidebar Component

**Files:**
- Create: `internal/ui/sidebar/sidebar.go`
- Create: `internal/ui/sidebar/sidebar_test.go`

- [ ] **Step 1: Write failing tests**

```go
package sidebar

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestNavigateDown(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.CursorIndex() != 1 {
		t.Errorf("cursor = %d, want 1", m.CursorIndex())
	}
}

func TestNavigateUp(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.CursorIndex() != 0 {
		t.Errorf("cursor = %d, want 0", m.CursorIndex())
	}
}

func TestSelectChannel(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}
}

func TestFilter(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
		{ID: "C3", Name: "engineering"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	visible := m.VisibleItems()
	if len(visible) != 2 {
		t.Errorf("expected 2 matches for 'gen', got %d", len(visible))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/sidebar/ -v`
Expected: compilation error

- [ ] **Step 3: Add bubble tea dependency**

Run: `go get github.com/charmbracelet/bubbletea`

- [ ] **Step 4: Implement sidebar.go**

```go
package sidebar

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type ChannelSelectedMsg struct {
	ID   string
	Name string
}

type Item struct {
	ID        string
	Name      string
	IsSection bool
	IsPrivate bool
	Unread    bool
}

type Model struct {
	items         []Item
	filtered      []int
	cursor        int
	filtering     bool
	filterText    string
	channelsOpen  bool
	dmsOpen       bool
	height        int
}

func New() Model {
	return Model{
		channelsOpen: true,
		dmsOpen:      true,
	}
}

func (m *Model) SetChannels(channels []slack.Channel) {
	m.items = nil
	m.items = append(m.items, Item{Name: "Channels", IsSection: true})
	for _, ch := range channels {
		prefix := "#"
		if ch.IsPrivate {
			prefix = "🔒"
		}
		m.items = append(m.items, Item{
			ID:        ch.ID,
			Name:      prefix + " " + ch.Name,
			IsPrivate: ch.IsPrivate,
		})
	}
	m.resetFilter()
}

func (m *Model) SetDMs(convs []slack.Conversation, resolve func(string) string) {
	m.items = append(m.items, Item{Name: "Direct Messages", IsSection: true})
	for _, c := range convs {
		name := resolve(c.UserID)
		m.items = append(m.items, Item{ID: c.ID, Name: name})
	}
	m.resetFilter()
}

func (m *Model) SetHeight(h int) { m.height = h }
func (m Model) CursorIndex() int { return m.cursor }

func (m Model) VisibleItems() []Item {
	var result []Item
	for _, idx := range m.filtered {
		result = append(result, m.items[idx])
	}
	return result
}

func (m Model) SelectedID() string {
	if m.cursor >= len(m.filtered) {
		return ""
	}
	return m.items[m.filtered[m.cursor]].ID
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			if m.items[m.filtered[m.cursor]].IsSection {
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
			}
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.items[m.filtered[m.cursor]].IsSection {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		}
	case "g":
		m.cursor = 0
		if len(m.filtered) > 0 && m.items[m.filtered[0]].IsSection {
			if len(m.filtered) > 1 {
				m.cursor = 1
			}
		}
	case "G":
		m.cursor = len(m.filtered) - 1
	case "enter":
		if m.cursor < len(m.filtered) {
			item := m.items[m.filtered[m.cursor]]
			if !item.IsSection {
				return m, func() tea.Msg {
					return ChannelSelectedMsg{ID: item.ID, Name: item.Name}
				}
			}
		}
	case "/":
		m.filtering = true
		m.filterText = ""
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filterText = ""
		m.resetFilter()
	case tea.KeyEnter:
		m.filtering = false
	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}
	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.applyFilter()
	}
	return m, nil
}

func (m *Model) resetFilter() {
	m.filtered = make([]int, len(m.items))
	for i := range m.items {
		m.filtered[i] = i
	}
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.resetFilter()
		return
	}
	m.filtered = nil
	query := strings.ToLower(m.filterText)
	for i, item := range m.items {
		if item.IsSection || strings.Contains(strings.ToLower(item.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	for i, idx := range m.filtered {
		if !m.items[idx].IsSection {
			m.cursor = i
			break
		}
	}
}

func (m Model) View() string {
	var b strings.Builder

	if m.filtering {
		b.WriteString(styles.SidebarSection.Render("/" + m.filterText + "█"))
		b.WriteString("\n")
	}

	for i, idx := range m.filtered {
		item := m.items[idx]
		if item.IsSection {
			b.WriteString(styles.SidebarSection.Render(item.Name))
		} else if i == m.cursor {
			b.WriteString(styles.SidebarSelected.Render(item.Name))
		} else {
			style := styles.SidebarNormal
			if item.Unread {
				style = style.Copy().Bold(true)
			}
			b.WriteString(style.Render(item.Name))
		}
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(30).Render(b.String())
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/sidebar/ -v`
Expected: all 4 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ui/sidebar/ go.mod go.sum
git commit -m "feat: add sidebar component with navigation and filtering"
```

---

### Task 11: Messages Component

**Files:**
- Create: `internal/ui/messages/messages.go`
- Create: `internal/ui/messages/messages_test.go`

- [ ] **Step 1: Write failing tests**

```go
package messages

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestSetMessages(t *testing.T) {
	m := New(nil)
	m.SetMessages([]slack.Message{
		{Text: "hello", Timestamp: "1706000001.000000", UserID: "U1"},
		{Text: "world", Timestamp: "1706000002.000000", UserID: "U2"},
	})
	if len(m.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(m.messages))
	}
}

func TestNavigateMessages(t *testing.T) {
	m := New(nil)
	m.SetMessages([]slack.Message{
		{Text: "first", Timestamp: "1706000001.000000"},
		{Text: "second", Timestamp: "1706000002.000000"},
		{Text: "third", Timestamp: "1706000003.000000"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestOpenThread(t *testing.T) {
	m := New(nil)
	m.SetMessages([]slack.Message{
		{Text: "has thread", Timestamp: "1706000001.000000", ThreadTS: "1706000001.000000", ReplyCount: 3},
	})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected command for thread open")
	}
}

func TestJumpToBottom(t *testing.T) {
	m := New(nil)
	m.SetMessages([]slack.Message{
		{Text: "a", Timestamp: "1"},
		{Text: "b", Timestamp: "2"},
		{Text: "c", Timestamp: "3"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/messages/ -v`
Expected: compilation error

- [ ] **Step 3: Implement messages.go**

```go
package messages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	islack "github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type OpenThreadMsg struct {
	ChannelID string
	ThreadTS  string
}

type OpenComposeMsg struct{}

type RequestPaginationMsg struct{}

type Model struct {
	messages  []islack.Message
	cursor    int
	channelID string
	channelName string
	width     int
	height    int
	renderer  *islack.Renderer
}

func New(renderer *islack.Renderer) Model {
	return Model{renderer: renderer}
}

func (m *Model) SetMessages(msgs []islack.Message) {
	m.messages = msgs
	if m.cursor >= len(msgs) {
		m.cursor = len(msgs) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) AppendMessages(msgs []islack.Message) {
	m.messages = append(m.messages, msgs...)
}

func (m *Model) SetChannel(id, name string) {
	m.channelID = id
	m.channelName = name
	m.messages = nil
	m.cursor = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) SelectedMessage() *islack.Message {
	if m.cursor >= len(m.messages) {
		return nil
	}
	return &m.messages[m.cursor]
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.messages)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor == 0 {
				return m, func() tea.Msg { return RequestPaginationMsg{} }
			}
		}
	case "g":
		m.cursor = 0
		return m, func() tea.Msg { return RequestPaginationMsg{} }
	case "G":
		m.cursor = len(m.messages) - 1
	case "t":
		if sel := m.SelectedMessage(); sel != nil {
			threadTS := sel.ThreadTS
			if threadTS == "" {
				threadTS = sel.Timestamp
			}
			return m, func() tea.Msg {
				return OpenThreadMsg{ChannelID: m.channelID, ThreadTS: threadTS}
			}
		}
	case "c":
		return m, func() tea.Msg { return OpenComposeMsg{} }
	}
	return m, nil
}

func (m Model) View() string {
	if m.channelID == "" {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("241")).
			Render("Select a channel")
	}

	var b strings.Builder
	header := styles.SidebarSection.Render(m.channelName)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	for i, msg := range m.messages {
		line := m.renderMessage(msg, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderMessage(msg islack.Message, selected bool) string {
	username := msg.Username
	if username == "" {
		username = msg.UserID
	}

	userStyle := styles.MessageUsername.Copy().
		Foreground(styles.ColorForUser(msg.UserID))

	ts := msg.Time().Format("15:04")

	text := msg.Text
	if m.renderer != nil {
		text = m.renderer.RenderPlain(text)
	}

	var extras []string
	if msg.ReplyCount > 0 {
		extras = append(extras, fmt.Sprintf("🧵%d", msg.ReplyCount))
	}
	for _, r := range msg.Reactions {
		extras = append(extras, styles.MessageReaction.Render(
			fmt.Sprintf("%s %d", r.Name, r.Count)))
	}

	line := userStyle.Render(username) + " " +
		styles.MessageTimestamp.Render(ts) + "\n" +
		styles.MessageText.Render(text)

	if len(extras) > 0 {
		line += "\n" + strings.Join(extras, " ")
	}

	if selected {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("63")).
			Render(line)
	}

	return lipgloss.NewStyle().PaddingLeft(2).Render(line)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/messages/ -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/messages/
git commit -m "feat: add messages component with navigation and thread opening"
```

---

### Task 12: Compose Overlay Component

**Files:**
- Create: `internal/ui/input/input.go`
- Create: `internal/ui/input/input_test.go`

- [ ] **Step 1: Write failing tests**

```go
package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTypeAndSend(t *testing.T) {
	m := New("C1", "", "#general")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if m.Value() != "hi" {
		t.Errorf("Value() = %q, want %q", m.Value(), "hi")
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected send command on Enter")
	}
}

func TestEscDismisses(t *testing.T) {
	m := New("C1", "", "#general")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected dismiss command on Esc")
	}
}

func TestEmptyEnterDoesNotSend(t *testing.T) {
	m := New("C1", "", "#general")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command on empty Enter")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/input/ -v`
Expected: compilation error

- [ ] **Step 3: Add textarea dependency**

Run: `go get github.com/charmbracelet/bubbles`

- [ ] **Step 4: Implement input.go**

```go
package input

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type SendMsg struct {
	ChannelID string
	ThreadTS  string
	Text      string
}

type DismissMsg struct{}

type Model struct {
	textarea    textarea.Model
	channelID   string
	threadTS    string
	channelName string
}

func New(channelID, threadTS, channelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetKeys("shift+enter", "alt+enter")

	return Model{
		textarea:    ta,
		channelID:   channelID,
		threadTS:    threadTS,
		channelName: channelName,
	}
}

func (m Model) Value() string {
	return m.textarea.Value()
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return DismissMsg{} }
		case tea.KeyEnter:
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				return SendMsg{
					ChannelID: m.channelID,
					ThreadTS:  m.threadTS,
					Text:      text,
				}
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	header := styles.SidebarSection.Render("Compose → " + m.channelName)
	return styles.OverlayStyle.Render(
		header + "\n" + m.textarea.View(),
	)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/input/ -v`
Expected: all 3 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ui/input/ go.mod go.sum
git commit -m "feat: add compose overlay component"
```

---

### Task 13: Thread Overlay Component

**Files:**
- Create: `internal/ui/thread/thread.go`
- Create: `internal/ui/thread/thread_test.go`

- [ ] **Step 1: Write failing tests**

```go
package thread

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestSetReplies(t *testing.T) {
	m := New(nil)
	m.SetReplies([]slack.Message{
		{Text: "parent", Timestamp: "1"},
		{Text: "reply1", Timestamp: "2"},
		{Text: "reply2", Timestamp: "3"},
	})
	if len(m.messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(m.messages))
	}
}

func TestNavigate(t *testing.T) {
	m := New(nil)
	m.SetReplies([]slack.Message{
		{Text: "parent", Timestamp: "1"},
		{Text: "reply", Timestamp: "2"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestEscCloses(t *testing.T) {
	m := New(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command on Esc")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/thread/ -v`
Expected: compilation error

- [ ] **Step 3: Implement thread.go**

```go
package thread

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	islack "github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type CloseMsg struct{}

type Model struct {
	messages  []islack.Message
	cursor    int
	channelID string
	threadTS  string
	width     int
	height    int
	renderer  *islack.Renderer
}

func New(renderer *islack.Renderer) Model {
	return Model{renderer: renderer}
}

func (m *Model) SetThread(channelID, threadTS string) {
	m.channelID = channelID
	m.threadTS = threadTS
	m.messages = nil
	m.cursor = 0
}

func (m *Model) SetReplies(msgs []islack.Message) {
	m.messages = msgs
	if m.cursor >= len(msgs) {
		m.cursor = len(msgs) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) ChannelID() string { return m.channelID }
func (m Model) ThreadTS() string  { return m.threadTS }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.messages)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			m.cursor = len(m.messages) - 1
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "c":
			return m, func() tea.Msg {
				return OpenThreadComposeMsg{
					ChannelID: m.channelID,
					ThreadTS:  m.threadTS,
				}
			}
		}
	}
	return m, nil
}

type OpenThreadComposeMsg struct {
	ChannelID string
	ThreadTS  string
}

func (m Model) View() string {
	var b strings.Builder

	header := fmt.Sprintf("Thread 🧵 (%d replies)", len(m.messages)-1)
	b.WriteString(styles.SidebarSection.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-4))
	b.WriteString("\n")

	for i, msg := range m.messages {
		username := msg.Username
		if username == "" {
			username = msg.UserID
		}
		userStyle := styles.MessageUsername.Copy().
			Foreground(styles.ColorForUser(msg.UserID))
		ts := msg.Time().Format("15:04")

		text := msg.Text
		if m.renderer != nil {
			text = m.renderer.RenderPlain(text)
		}

		line := userStyle.Render(username) + " " +
			styles.MessageTimestamp.Render(ts) + "\n" +
			styles.MessageText.Render(text)

		if i == 0 {
			b.WriteString(lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("241")).
				MarginBottom(1).
				Render(line))
		} else if i == m.cursor {
			b.WriteString(lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("63")).
				Render(line))
		} else {
			b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(line))
		}
		b.WriteString("\n")
	}

	return styles.ThreadBorder.Width(m.width).Height(m.height).Render(b.String())
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/thread/ -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/thread/
git commit -m "feat: add thread overlay component"
```

---

### Task 14: Help Overlay Component

**Files:**
- Create: `internal/ui/help/help.go`

- [ ] **Step 1: Implement help.go**

```go
package help

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type CloseMsg struct{}

type Model struct {
	width  int
	height int
}

func New() Model { return Model{} }

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.Type == tea.KeyEsc || msg.String() == "?" || msg.String() == "q" {
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	bindings := []struct{ key, action string }{
		{"j/k", "Navigate up/down"},
		{"h/l", "Focus sidebar/messages"},
		{"Enter", "Select channel"},
		{"c", "Compose message"},
		{"t", "Open thread"},
		{"r", "React with emoji"},
		{"y", "Copy message"},
		{"/", "Filter channels"},
		{"g/G", "Jump to top/bottom"},
		{"Tab", "Toggle sidebar section"},
		{"Esc", "Close overlay"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	var b strings.Builder
	b.WriteString(styles.SidebarSection.Render("Keybindings"))
	b.WriteString("\n\n")

	for _, bind := range bindings {
		b.WriteString(styles.StatusBarKey.Render(" "+bind.key+" "))
		b.WriteString("  ")
		b.WriteString(bind.action)
		b.WriteString("\n")
	}

	return styles.OverlayStyle.Width(m.width).Render(b.String())
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/ui/help/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/ui/help/
git commit -m "feat: add help overlay component"
```

---

### Task 15: Emoji Picker Component

**Files:**
- Create: `internal/ui/emoji/emoji.go`
- Create: `internal/ui/emoji/emoji_test.go`

- [ ] **Step 1: Write failing tests**

```go
package emoji

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilter(t *testing.T) {
	m := New("C1", "1706000001.000000")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	visible := m.VisibleEmojis()
	for _, e := range visible {
		if e.Name == "+1" || e.Name == "thumbsup" {
			return
		}
	}
	if len(visible) == 0 {
		t.Error("expected at least one thumbs emoji match")
	}
}

func TestSelectEmoji(t *testing.T) {
	m := New("C1", "1706000001.000000")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}
}

func TestEscCloses(t *testing.T) {
	m := New("C1", "1706000001.000000")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command on Esc")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/emoji/ -v`
Expected: compilation error

- [ ] **Step 3: Implement emoji.go**

```go
package emoji

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type SelectMsg struct {
	ChannelID string
	Timestamp string
	Emoji     string
}

type CloseMsg struct{}

type EmojiEntry struct {
	Char string
	Name string
}

type Model struct {
	channelID string
	timestamp string
	filter    string
	cursor    int
	emojis    []EmojiEntry
}

func New(channelID, timestamp string) Model {
	return Model{
		channelID: channelID,
		timestamp: timestamp,
		emojis:    commonEmojis(),
	}
}

func (m Model) VisibleEmojis() []EmojiEntry {
	if m.filter == "" {
		return m.emojis
	}
	query := strings.ToLower(m.filter)
	var result []EmojiEntry
	for _, e := range m.emojis {
		if strings.Contains(e.Name, query) {
			result = append(result, e)
		}
	}
	return result
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return CloseMsg{} }
		case tea.KeyEnter:
			visible := m.VisibleEmojis()
			if m.cursor < len(visible) {
				emoji := visible[m.cursor]
				return m, func() tea.Msg {
					return SelectMsg{
						ChannelID: m.channelID,
						Timestamp: m.timestamp,
						Emoji:     emoji.Name,
					}
				}
			}
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
			}
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			visible := m.VisibleEmojis()
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(styles.SidebarSection.Render("React: " + m.filter + "█"))
	b.WriteString("\n\n")

	visible := m.VisibleEmojis()
	limit := 10
	if len(visible) < limit {
		limit = len(visible)
	}

	for i := 0; i < limit; i++ {
		e := visible[i]
		if i == m.cursor {
			b.WriteString(styles.SidebarSelected.Render(e.Char + " " + e.Name))
		} else {
			b.WriteString(styles.SidebarNormal.Render(e.Char + " " + e.Name))
		}
		b.WriteString("\n")
	}

	return styles.OverlayStyle.Render(b.String())
}

func commonEmojis() []EmojiEntry {
	return []EmojiEntry{
		{"👍", "+1"}, {"👎", "-1"}, {"❤️", "heart"}, {"😂", "joy"},
		{"🎉", "tada"}, {"🙏", "pray"}, {"🔥", "fire"}, {"👀", "eyes"},
		{"✅", "white_check_mark"}, {"❌", "x"}, {"💯", "100"},
		{"🚀", "rocket"}, {"👏", "clap"}, {"🤔", "thinking_face"},
		{"😍", "heart_eyes"}, {"😭", "sob"}, {"😅", "sweat_smile"},
		{"🙌", "raised_hands"}, {"💪", "muscle"}, {"⭐", "star"},
		{"📝", "memo"}, {"🐛", "bug"}, {"💡", "bulb"}, {"⚡", "zap"},
		{"🎯", "dart"}, {"🔧", "wrench"}, {"📢", "loudspeaker"},
		{"💬", "speech_balloon"}, {"👋", "wave"}, {"🤝", "handshake"},
		{"thumbsup", "thumbsup"}, {"thumbsdown", "thumbsdown"},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/emoji/ -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/emoji/
git commit -m "feat: add emoji picker with fuzzy search"
```

---

### Task 16: Root App Model + Keymap

**Files:**
- Create: `internal/app/keys.go`
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

```go
package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFocusSwitching(t *testing.T) {
	m := newTestApp()

	if m.focus != focusSidebar {
		t.Errorf("initial focus = %d, want sidebar", m.focus)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focus != focusMessages {
		t.Errorf("after 'l', focus = %d, want messages", m.focus)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focus != focusSidebar {
		t.Errorf("after 'h', focus = %d, want sidebar", m.focus)
	}
}

func TestQuit(t *testing.T) {
	m := newTestApp()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestHelpToggle(t *testing.T) {
	m := newTestApp()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.showHelp {
		t.Error("expected help to be shown")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.showHelp {
		t.Error("expected help to be hidden")
	}
}

func newTestApp() Model {
	return New(nil, "testcorp")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/ -v`
Expected: compilation error

- [ ] **Step 3: Implement keys.go**

```go
package app

type focusArea int

const (
	focusSidebar focusArea = iota
	focusMessages
	focusThread
)
```

- [ ] **Step 4: Implement app.go**

```go
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/emoji"
	"github.com/filipeestacio/lazyslack/internal/ui/help"
	"github.com/filipeestacio/lazyslack/internal/ui/input"
	"github.com/filipeestacio/lazyslack/internal/ui/messages"
	"github.com/filipeestacio/lazyslack/internal/ui/sidebar"
	"github.com/filipeestacio/lazyslack/internal/ui/statusbar"
	"github.com/filipeestacio/lazyslack/internal/ui/thread"
)

type Model struct {
	client    slack.SlackClient
	workspace string
	width     int
	height    int

	focus     focusArea
	showHelp  bool
	composing bool
	showEmoji bool
	showThread bool

	sidebar   sidebar.Model
	messages  messages.Model
	thread    thread.Model
	input     input.Model
	emoji     emoji.Model
	help      help.Model
	statusbar statusbar.Model
	renderer  *slack.Renderer
}

func New(client slack.SlackClient, workspace string) Model {
	r := slack.NewRenderer(nil)
	return Model{
		client:    client,
		workspace: workspace,
		focus:     focusSidebar,
		sidebar:   sidebar.New(),
		messages:  messages.New(r),
		thread:    thread.New(r),
		help:      help.New(),
		statusbar: statusbar.New(workspace),
		renderer:  r,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadChannels
}

func (m Model) loadChannels() tea.Msg {
	if m.client == nil {
		return nil
	}
	channels, err := m.client.ListChannels()
	if err != nil {
		return errMsg{err}
	}
	return channelsLoadedMsg{channels}
}

type channelsLoadedMsg struct{ channels []slack.Channel }
type dmsLoadedMsg struct{ convs []slack.Conversation }
type historyLoadedMsg struct{ result *slack.HistoryResult }
type threadLoadedMsg struct{ messages []slack.Message }
type messageSentMsg struct{}
type reactionAddedMsg struct{}
type errMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		sidebarWidth := 30
		msgWidth := m.width - sidebarWidth
		m.messages.SetSize(msgWidth, m.height-2)
		m.thread.SetSize(msgWidth/2, m.height-4)
		m.help.SetSize(50, m.height-4)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case channelsLoadedMsg:
		m.sidebar.SetChannels(msg.channels)
		return m, m.loadDMs

	case dmsLoadedMsg:
		m.sidebar.SetDMs(msg.convs, func(id string) string { return id })
		return m, nil

	case historyLoadedMsg:
		if msg.result != nil {
			m.messages.SetMessages(msg.result.Messages)
		}
		return m, nil

	case threadLoadedMsg:
		m.thread.SetReplies(msg.messages)
		return m, nil

	case sidebar.ChannelSelectedMsg:
		m.messages.SetChannel(msg.ID, msg.Name)
		return m, m.fetchHistory(msg.ID)

	case messages.OpenThreadMsg:
		m.showThread = true
		m.focus = focusThread
		m.thread.SetThread(msg.ChannelID, msg.ThreadTS)
		return m, m.fetchThread(msg.ChannelID, msg.ThreadTS)

	case messages.OpenComposeMsg:
		m.composing = true
		sel := m.messages.SelectedMessage()
		channelName := ""
		if sel != nil {
			channelName = "message"
		}
		m.input = input.New(m.sidebar.SelectedID(), "", channelName)
		return m, m.input.Init()

	case thread.CloseMsg:
		m.showThread = false
		m.focus = focusMessages
		return m, nil

	case thread.OpenThreadComposeMsg:
		m.composing = true
		m.input = input.New(msg.ChannelID, msg.ThreadTS, "thread")
		return m, m.input.Init()

	case input.SendMsg:
		m.composing = false
		return m, m.sendMessage(msg)

	case input.DismissMsg:
		m.composing = false
		return m, nil

	case emoji.SelectMsg:
		m.showEmoji = false
		return m, m.addReaction(msg)

	case emoji.CloseMsg:
		m.showEmoji = false
		return m, nil

	case help.CloseMsg:
		m.showHelp = false
		return m, nil

	case messageSentMsg:
		return m, nil

	case errMsg:
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.composing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if m.showEmoji {
		var cmd tea.Cmd
		m.emoji, cmd = m.emoji.Update(msg)
		return m, cmd
	}

	if m.showHelp {
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "h", "left":
		m.focus = focusSidebar
		return m, nil
	case "l", "right":
		if m.showThread {
			m.focus = focusThread
		} else {
			m.focus = focusMessages
		}
		return m, nil
	case "r":
		if m.focus == focusMessages {
			if sel := m.messages.SelectedMessage(); sel != nil {
				m.showEmoji = true
				m.emoji = emoji.New(m.sidebar.SelectedID(), sel.Timestamp)
				return m, nil
			}
		}
	}

	switch m.focus {
	case focusSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd
	case focusMessages:
		var cmd tea.Cmd
		m.messages, cmd = m.messages.Update(msg)
		return m, cmd
	case focusThread:
		var cmd tea.Cmd
		m.thread, cmd = m.thread.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) fetchHistory(channelID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}
		result, err := m.client.GetHistory(channelID, "")
		if err != nil {
			return errMsg{err}
		}
		return historyLoadedMsg{result}
	}
}

func (m Model) fetchThread(channelID, threadTS string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}
		msgs, err := m.client.GetThreadReplies(channelID, threadTS)
		if err != nil {
			return errMsg{err}
		}
		return threadLoadedMsg{msgs}
	}
}

func (m Model) loadDMs() tea.Msg {
	if m.client == nil {
		return nil
	}
	convs, err := m.client.ListDMs()
	if err != nil {
		return errMsg{err}
	}
	return dmsLoadedMsg{convs}
}

func (m Model) sendMessage(msg input.SendMsg) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return messageSentMsg{}
		}
		var err error
		if msg.ThreadTS != "" {
			err = m.client.ReplyToThread(msg.ChannelID, msg.ThreadTS, msg.Text)
		} else {
			err = m.client.SendMessage(msg.ChannelID, msg.Text)
		}
		if err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}

func (m Model) addReaction(msg emoji.SelectMsg) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return reactionAddedMsg{}
		}
		err := m.client.AddReaction(msg.ChannelID, msg.Timestamp, msg.Emoji)
		if err != nil {
			return errMsg{err}
		}
		return reactionAddedMsg{}
	}
}

func (m Model) View() string {
	sidebarView := m.sidebar.View()
	messagesView := m.messages.View()

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, messagesView)

	if m.showThread {
		threadView := m.thread.View()
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarView,
			lipgloss.NewStyle().Width(m.width-30-m.width/2).Render(messagesView),
			threadView,
		)
	}

	if m.composing {
		mainArea = lipgloss.Place(m.width, m.height-1,
			lipgloss.Center, lipgloss.Bottom,
			m.input.View(),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.showEmoji {
		mainArea = lipgloss.Place(m.width, m.height-1,
			lipgloss.Center, lipgloss.Center,
			m.emoji.View(),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.showHelp {
		mainArea = lipgloss.Place(m.width, m.height-1,
			lipgloss.Center, lipgloss.Center,
			m.help.View(),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	statusView := m.statusbar.View(m.width)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, statusView)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/app/ -v`
Expected: all 3 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/
git commit -m "feat: add root app model with focus routing and panel composition"
```

---

### Task 17: Main Entrypoint

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Implement main.go**

```go
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/app"
	"github.com/filipeestacio/lazyslack/internal/auth"
	"github.com/filipeestacio/lazyslack/internal/config"
	islack "github.com/filipeestacio/lazyslack/internal/slack"
)

func main() {
	debug := len(os.Args) > 1 && os.Args[1] == "--debug"

	if debug {
		f, err := os.OpenFile("lazyslack.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Auth.Method == "oauth" && cfg.Auth.OAuthClientID != "" && cfg.Auth.Token == "" {
		scopes := []string{
			"channels:read", "channels:history", "chat:write", "reactions:write",
			"users:read", "im:read", "im:history", "mpim:read", "mpim:history",
			"groups:read", "groups:history",
		}
		token, err := auth.RunOAuthFlow(cfg.Auth.OAuthClientID, cfg.Auth.OAuthClientSecret, scopes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "OAuth failed: %v\n", err)
			os.Exit(1)
		}
		cfg.Auth.Token = token
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	if cfg.Auth.Token == "" {
		fmt.Println("No auth token configured.")
		fmt.Println("Edit ~/.config/lazyslack/config.yaml with your auth details.")
		fmt.Println()
		fmt.Println("Session token method:")
		fmt.Println("  1. Open Slack in your browser")
		fmt.Println("  2. Open Developer Tools (F12)")
		fmt.Println("  3. Go to Application > Cookies, copy the 'd' cookie value")
		fmt.Println("  4. In Network tab, find any API call, copy the 'Authorization: Bearer xoxc-...' token")
		fmt.Println()
		fmt.Println("OAuth method:")
		fmt.Println("  1. Create a Slack app at https://api.slack.com/apps")
		fmt.Println("  2. Add user token scopes: channels:read, channels:history, chat:write,")
		fmt.Println("     reactions:write, users:read, im:read, im:history, mpim:read,")
		fmt.Println("     mpim:history, groups:read, groups:history")
		fmt.Println("  3. Set oauth_client_id and oauth_client_secret in config.yaml")
		fmt.Println("  4. Run lazyslack again to complete the OAuth flow")
		os.Exit(0)
	}

	info, err := auth.ValidateToken(cfg, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Auth failed: %v\nPlease check your token in ~/.config/lazyslack/config.yaml\n", err)
		os.Exit(1)
	}

	workspace := info.Team
	if cfg.Workspace.Name != "" {
		workspace = cfg.Workspace.Name
	}

	client := islack.NewClient(cfg.Auth.Token, cfg.Auth.Cookie)
	model := app.New(client, workspace)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build .`
Expected: binary created successfully

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add main entrypoint with auth setup and TUI launch"
```

---

### Task 18: Integration — Polling Wired Into App

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Write failing test for polling integration**

Add to `internal/app/app_test.go`:

```go
func TestNewMessagesUpdateView(t *testing.T) {
	m := newTestApp()
	m.messages.SetChannel("C1", "#test")

	newMsgs := []slack.Message{
		{Text: "new msg", Timestamp: "1706000010.000000", UserID: "U1"},
	}

	m, _ = m.Update(newMessagesMsg{messages: newMsgs})
	if sel := m.messages.SelectedMessage(); sel == nil || sel.Text != "new msg" {
		t.Error("expected new message to appear in messages view")
	}
}
```

- [ ] **Step 2: Run tests to verify it fails**

Run: `go test ./internal/app/ -v -run TestNewMessages`
Expected: compilation error — `newMessagesMsg` not defined

- [ ] **Step 3: Add polling message type and handler to app.go**

Add to the msg types:
```go
type newMessagesMsg struct{ messages []slack.Message }
```

Add to the `Update` switch:
```go
case newMessagesMsg:
    m.messages.SetMessages(msg.messages)
    return m, nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/ -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/
git commit -m "feat: wire polling messages into app model"
```

---

### Task 19: Update Nix Flake with vendorHash

**Files:**
- Modify: `flake.nix`

- [ ] **Step 1: Tidy go modules**

Run: `go mod tidy`

- [ ] **Step 2: Test the build with nix**

Run: `nix build` — this will fail with a hash mismatch. Copy the expected hash from the error.

- [ ] **Step 3: Update vendorHash in flake.nix**

Replace `vendorHash = null;` with the hash from the nix build error output. If the project uses modules without a vendor directory, use `vendorHash = null;` and rely on `goModules` or set the correct hash.

- [ ] **Step 4: Verify nix build succeeds**

Run: `nix build`
Expected: `./result/bin/lazyslack` binary produced

- [ ] **Step 5: Commit**

```bash
git add flake.nix go.mod go.sum
git commit -m "feat: finalize nix flake with correct vendor hash"
```

---

### Task 20: User Cache + Resolver Wiring

**Files:**
- Create: `internal/slack/usercache.go`
- Create: `internal/slack/usercache_test.go`
- Modify: `internal/app/app.go`

- [ ] **Step 1: Write failing tests**

```go
package slack

import "testing"

type mockUserFetcher struct {
	users []User
}

func (m *mockUserFetcher) GetUsers() ([]User, error) {
	return m.users, nil
}

func TestUserCacheResolve(t *testing.T) {
	fetcher := &mockUserFetcher{
		users: []User{
			{ID: "U1", Name: "alice", DisplayName: "Alice A"},
			{ID: "U2", Name: "bob", DisplayName: "Bob B"},
		},
	}
	cache := NewUserCache(fetcher)
	if err := cache.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got := cache.ResolveUser("U1"); got != "Alice A" {
		t.Errorf("ResolveUser(U1) = %q, want %q", got, "Alice A")
	}
	if got := cache.ResolveUser("U999"); got != "U999" {
		t.Errorf("ResolveUser(U999) = %q, want %q", got, "U999")
	}
}

func TestUserCacheResolveChannel(t *testing.T) {
	cache := NewUserCache(nil)
	cache.SetChannels([]Channel{{ID: "C1", Name: "general"}, {ID: "C2", Name: "random"}})

	if got := cache.ResolveChannel("C1"); got != "general" {
		t.Errorf("ResolveChannel(C1) = %q, want %q", got, "general")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/slack/ -v -run "TestUserCache"`
Expected: compilation error

- [ ] **Step 3: Implement usercache.go**

```go
package slack

import "sync"

type UserFetcher interface {
	GetUsers() ([]User, error)
}

type UserCache struct {
	fetcher  UserFetcher
	mu       sync.RWMutex
	users    map[string]User
	channels map[string]string
}

func NewUserCache(fetcher UserFetcher) *UserCache {
	return &UserCache{
		fetcher:  fetcher,
		users:    make(map[string]User),
		channels: make(map[string]string),
	}
}

func (c *UserCache) Load() error {
	if c.fetcher == nil {
		return nil
	}
	users, err := c.fetcher.GetUsers()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, u := range users {
		c.users[u.ID] = u
	}
	return nil
}

func (c *UserCache) SetChannels(channels []Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range channels {
		c.channels[ch.ID] = ch.Name
	}
}

func (c *UserCache) ResolveUser(id string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if u, ok := c.users[id]; ok {
		if u.DisplayName != "" {
			return u.DisplayName
		}
		return u.Name
	}
	return id
}

func (c *UserCache) ResolveChannel(id string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if name, ok := c.channels[id]; ok {
		return name
	}
	return id
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/slack/ -v -run "TestUserCache"`
Expected: all 2 tests PASS

- [ ] **Step 5: Wire UserCache into App.Init**

In `internal/app/app.go`, update `New()` to accept and store a `*slack.UserCache`. Update `Init()` to load users on startup and pass the cache as the `UserResolver` to `NewRenderer`. Update the sidebar `SetDMs` call to use `cache.ResolveUser` instead of the identity function.

- [ ] **Step 6: Update main.go**

Create the `UserCache` after client initialization and pass it to `app.New`:
```go
userCache := islack.NewUserCache(client)
model := app.New(client, userCache, workspace)
```

- [ ] **Step 7: Commit**

```bash
git add internal/slack/usercache.go internal/slack/usercache_test.go internal/app/app.go main.go
git commit -m "feat: add user cache and wire resolver into renderer and sidebar"
```

---

### Task 21: Clipboard (y) and Sidebar Tab Toggle

**Files:**
- Modify: `internal/ui/messages/messages.go`
- Modify: `internal/ui/sidebar/sidebar.go`
- Modify: `internal/ui/sidebar/sidebar_test.go`
- Modify: `internal/ui/messages/messages_test.go`

- [ ] **Step 1: Add clipboard test for messages**

Add to `internal/ui/messages/messages_test.go`:

```go
func TestYankMessage(t *testing.T) {
	m := New(nil)
	m.SetMessages([]slack.Message{
		{Text: "copy me", Timestamp: "1706000001.000000"},
	})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected command for yank")
	}
}
```

- [ ] **Step 2: Add Tab toggle test for sidebar**

Add to `internal/ui/sidebar/sidebar_test.go`:

```go
func TestTabTogglesSection(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
	})

	// Cursor is on first non-section item; move to section header
	// Tab should toggle the section collapse state
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// After toggling, the channels section should be collapsed
	if m.channelsOpen {
		t.Error("expected channels section to be collapsed after Tab")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/ui/messages/ ./internal/ui/sidebar/ -v -run "TestYank|TestTab"`
Expected: failures

- [ ] **Step 4: Add clipboard dependency**

Run: `go get golang.design/x/clipboard`

- [ ] **Step 5: Implement `y` in messages.go**

Add a `CopyMsg` type and handle `y` in `handleKey`:
```go
type CopyMsg struct{ Text string }
```
In the `"y"` case:
```go
case "y":
    if sel := m.SelectedMessage(); sel != nil {
        text := sel.Text
        return m, func() tea.Msg { return CopyMsg{Text: text} }
    }
```

Handle `CopyMsg` in `app.go` to write to clipboard using `golang.design/x/clipboard`.

- [ ] **Step 6: Implement `Tab` in sidebar.go**

Add `"tab"` case to `updateNormal`:
```go
case "tab":
    // Find which section the cursor is under and toggle it
    for i := m.filtered[m.cursor]; i >= 0; i-- {
        if m.items[i].IsSection {
            if m.items[i].Name == "Channels" {
                m.channelsOpen = !m.channelsOpen
            } else {
                m.dmsOpen = !m.dmsOpen
            }
            m.rebuildFiltered()
            break
        }
    }
```

Add `rebuildFiltered()` method that excludes items in collapsed sections.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/ui/messages/ ./internal/ui/sidebar/ -v -run "TestYank|TestTab"`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/ui/messages/ internal/ui/sidebar/ internal/app/app.go go.mod go.sum
git commit -m "feat: add clipboard yank and sidebar section toggle"
```

---

### Task 22: End-to-End Smoke Test

**Files:** None (manual verification)

- [ ] **Step 1: Build and run**

Run: `go build -o lazyslack . && ./lazyslack`
Expected: if no token configured, shows setup instructions and exits cleanly

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: all tests pass across all packages

- [ ] **Step 3: Run linter**

Run: `golangci-lint run ./...`
Expected: no errors (warnings acceptable)

- [ ] **Step 4: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address lint and test issues from smoke test"
```
