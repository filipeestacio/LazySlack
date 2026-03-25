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
