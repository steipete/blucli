package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_MissingFileReturnsEmptyAliases(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := Load(LoadOptions{Path: path})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if cfg.Aliases == nil || len(cfg.Aliases) != 0 {
		t.Fatalf("aliases=%v; want empty map", cfg.Aliases)
	}
}

func TestLoad_InvalidJSONErrors(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(LoadOptions{Path: path}); err == nil {
		t.Fatalf("want error")
	}
}

func TestLoad_EnsuresAliasesMap(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"default_device":"x"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(LoadOptions{Path: path})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if cfg.Aliases == nil {
		t.Fatalf("aliases=nil")
	}
}

func TestSaveConfig_CreatesParentDir(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg := Config{
		DefaultDevice: "http://127.0.0.1:11000",
		Aliases:       map[string]string{"k": "v"},
		Spotify: SpotifyConfig{
			ClientID: "CID",
			Token: SpotifyToken{
				AccessToken:  "AT",
				RefreshToken: "RT",
				ExpiresAt:    time.Now().Add(time.Hour),
				TokenType:    "Bearer",
				Scope:        "s",
			},
		},
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("err = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "\"default_device\"") || !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("config = %q", string(data))
	}
}

func TestConfigPath_Explicit(t *testing.T) {
	t.Parallel()

	if got, err := ConfigPath("/tmp/x.json"); err != nil || got != "/tmp/x.json" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestPaths_ReturnsBluPaths(t *testing.T) {
	t.Parallel()

	p, err := Paths()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(p.ConfigPath, string(filepath.Separator)+"blu"+string(filepath.Separator)) {
		t.Fatalf("ConfigPath=%q", p.ConfigPath)
	}
	if !strings.Contains(p.CachePath, string(filepath.Separator)+"blu"+string(filepath.Separator)) {
		t.Fatalf("CachePath=%q", p.CachePath)
	}
}

func TestConfigPath_Default(t *testing.T) {
	t.Parallel()

	p, err := configPath("")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if p == "" || !strings.Contains(p, string(filepath.Separator)+"blu"+string(filepath.Separator)) {
		t.Fatalf("path=%q", p)
	}
}

func TestSaveConfig_ParentDirError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := filepath.Join(blocker, "config.json")
	if err := SaveConfig(path, Config{Aliases: map[string]string{}}); err == nil {
		t.Fatalf("want error")
	}
}
