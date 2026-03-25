# LazySlack — Design Spec

A terminal UI for Slack, inspired by LazyGit and LazyDocker. Keyboard-driven with mouse support. Single-workspace, conversational feature set (read, write, reply, react).

## Stack

- **Language:** Go
- **TUI framework:** Bubble Tea (Elm architecture: Model → Update → View)
- **Styling:** Lip Gloss
- **Slack API:** slack-go/slack
- **Config format:** YAML
- **Dev environment:** Nix (flake.nix)

## Authentication

Session token authentication only. The rest of the app is auth-agnostic.

### Session Token (xoxc + cookie)

- User extracts `xoxc-*` token and `d` cookie from browser dev tools
- First-run CLI shows step-by-step extraction instructions
- Validated on startup via `auth.test`; on expiry, prompt re-authentication
- Stored in config file with 0600 permissions

### Config Structure

```yaml
auth:
  method: session_token
  token: xoxc-...
  cookie: "d=xoxd-..."
workspace:
  name: mycompany
```

Config path: `~/.config/lazyslack/config.yaml`
State/cache path: `~/.local/state/lazyslack/`

## Architecture

```
┌─────────────────────────────────────┐
│            App (root model)         │
│  ┌──────────┐ ┌──────────────────┐  │
│  │ Sidebar  │ │   Messages       │  │
│  │  Model   │ │    Model         │  │
│  └──────────┘ └──────────────────┘  │
│               ┌──────────────────┐  │
│               │ Thread Overlay   │  │
│               │    Model         │  │
│               └──────────────────┘  │
│  ┌──────────────────────────────────┤
│  │ Compose Overlay                  │
│  └──────────────────────────────────┤
│  ┌──────────────────────────────────┤
│  │ Status Bar                       │
│  └──────────────────────────────────┤
├─────────────────────────────────────┤
│         Slack Client (API layer)    │
├─────────────────────────────────────┤
│         Config / Auth               │
└─────────────────────────────────────┘
```

Three layers:

- **UI layer** — Bubble Tea models for each panel, rendered with Lip Gloss
- **Client layer** — wraps slack-go/slack, unified interface for both auth methods
- **Config layer** — YAML config, XDG paths, auth token management

Communication between panels via Bubble Tea messages. Root model routes messages and manages focus.

## Slack Client Interface

```go
type SlackClient interface {
    AuthTest() (*AuthInfo, error)
    ListChannels() ([]Channel, error)
    ListDMs() ([]Conversation, error)
    GetHistory(channelID string, cursor string) ([]Message, error)
    GetThreadReplies(channelID, threadTS string) ([]Message, error)
    SendMessage(channelID, text string) error
    ReplyToThread(channelID, threadTS, text string) error
    AddReaction(channelID, timestamp, emoji string) error
    RemoveReaction(channelID, timestamp, emoji string) error
    GetUsers() ([]User, error)
}
```

### Polling

- Background goroutine polls `GetHistory` for the active channel every 3 seconds (configurable)
- New messages dispatched as `NewMessagesMsg` Bubble Tea commands
- Channel switch triggers an immediate fetch; polling resumes on the new channel's cadence
- Polling pauses when no channel is selected
- Exponential backoff on transient errors

### Rate Limiting

- Token bucket on top of slack-go's built-in handling
- Respect `Retry-After` headers
- Status bar indicator when rate-limited

### User Cache

- Workspace user list fetched on startup, cached in memory
- Resolves user IDs to display names in messages
- Refresh every 30 minutes or on cache miss

## UI Layout

Two-panel default with overlays:

- **Sidebar (left)** — channel/DM list, collapsible sections
- **Messages (center)** — message feed for selected channel, takes remaining width
- **Thread overlay** — floating panel over right half of messages area, appears on demand
- **Compose overlay** — message input, appears on `c`, captures all keyboard input
- **Status bar (bottom)** — workspace name, connection status, keybind hints

## Keybindings

Vim-style single-key commands. All shortcuts active in normal mode. Compose overlay captures all input until sent or dismissed.

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Focus sidebar |
| `l` / `→` | Focus messages |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Tab` | Collapse/expand sidebar section |
| `Enter` | Select channel (in sidebar) |
| `/` | Filter/search channels (in sidebar) |

### Actions

| Key | Action |
|-----|--------|
| `c` | Open compose overlay (send message / reply in thread) |
| `t` | Open thread overlay for selected message |
| `r` | Open emoji reaction picker (standard Unicode emoji, fuzzy-searchable) |
| `y` | Copy message text to clipboard |
| `?` | Show help overlay (all keybindings) |
| `Esc` | Close overlay / cancel compose |
| `Enter` | Send message (in compose overlay) |
| `Shift+Enter` | Newline (in compose overlay) |

### Compose Overlay Behavior

- `c` opens the overlay; all keypresses route to text input
- `Enter` sends the message, closes overlay, returns to normal mode
- `Esc` dismisses without sending, returns to normal mode
- Context-aware: in messages panel, composes to channel; in thread overlay, replies to thread

## Message Rendering

Slack markup converted to styled terminal output:

- `*bold*` → bold
- `_italic_` → italic
- `~strike~` → strikethrough
- `` `code` `` → inline code style
- `<@U123>` → resolved username from cache
- `<#C123>` → resolved channel name
- Common cases handled; unrecognized markup passed through as plain text

## Error Handling

- **Network failures:** Exponential backoff, "disconnected" status bar indicator after 5 consecutive failures, silent retry continues
- **Auth expiry:** Stop polling, surface "Session expired — press `A` to re-authenticate"
- **Rate limiting:** Respect `Retry-After`, status bar indicator
- **Large channels:** Load one page (50 messages) initially, paginate on scroll-up
- **Graceful degradation:** Features unavailable with current auth method are hidden (no keybinding shown)

## Testing Strategy

- **Unit tests:** Slack client (mock HTTP), config parsing, auth flows, message renderer
- **Integration tests:** Each UI model tested via Bubble Tea's `Update(msg)` → assert model state
- **Debug mode:** `--debug` flag dumps API requests/responses to log file
- No end-to-end TUI tests — manual testing against real workspace covers visual integration

## Project Structure

```
lazyslack/
├── flake.nix
├── flake.lock
├── go.mod
├── go.sum
├── main.go
├── internal/
│   ├── app/
│   ├── ui/
│   │   ├── sidebar/
│   │   ├── messages/
│   │   ├── thread/
│   │   ├── input/
│   │   ├── emoji/
│   │   ├── help/
│   │   └── styles/
│   ├── slack/
│   ├── auth/
│   └── config/
└── docs/
```

## Nix Dev Environment

`flake.nix` provides a dev shell with:

- Go toolchain
- gopls (language server)
- golangci-lint
- Optionally: a build derivation for the final binary

`nix develop` drops into a ready-to-go environment.

## Scope Boundaries (v1)

**In scope:** Browse channels/DMs, read messages, send messages, reply to threads, react with emoji, single workspace, polling-based updates.

**Out of scope (v1):** OAuth (requires app installation and typically admin approval), multi-workspace, WebSocket real-time, file uploads, search, notifications, channel management, user presence, typing indicators.
