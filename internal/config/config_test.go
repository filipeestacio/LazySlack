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
