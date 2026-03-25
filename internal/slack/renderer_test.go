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
