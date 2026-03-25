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
