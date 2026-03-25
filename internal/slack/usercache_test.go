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
