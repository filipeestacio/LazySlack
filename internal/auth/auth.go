package auth

import (
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
