package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveConfigRestrictsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions")
	}
	for _, existing := range []bool{false, true} {
		name := "new"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if existing {
				if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cfg := Config{DefaultDevice: "synthetic-player", Spotify: SpotifyConfig{Token: SpotifyToken{AccessToken: "synthetic-token"}}}
			if err := SaveConfig(path, cfg); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("config permissions = %o; want 600", info.Mode().Perm())
			}
			loaded, err := Load(LoadOptions{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if loaded.DefaultDevice != cfg.DefaultDevice || loaded.Spotify.Token.AccessToken != cfg.Spotify.Token.AccessToken {
				t.Fatal("config contents changed")
			}
		})
	}
}
