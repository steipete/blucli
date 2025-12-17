package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DefaultDevice string            `json:"default_device,omitempty"`
	Aliases       map[string]string `json:"aliases,omitempty"`
	Spotify       SpotifyConfig     `json:"spotify,omitempty"`
}

type SpotifyConfig struct {
	ClientID string       `json:"client_id,omitempty"`
	Token    SpotifyToken `json:"token,omitempty"`
}

type SpotifyToken struct {
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

type LoadOptions struct {
	Path string
}

func Load(opts LoadOptions) (Config, error) {
	path, err := configPath(opts.Path)
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{Aliases: map[string]string{}}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]string{}
	}
	return cfg, nil
}

type PathSet struct {
	ConfigPath string
	CachePath  string
}

func Paths() (PathSet, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return PathSet{}, err
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return PathSet{}, err
	}

	return PathSet{
		ConfigPath: filepath.Join(configDir, "blu", "config.json"),
		CachePath:  filepath.Join(cacheDir, "blu", "discovery.json"),
	}, nil
}

func configPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	paths, err := Paths()
	if err != nil {
		return "", err
	}
	return paths.ConfigPath, nil
}

func ConfigPath(explicit string) (string, error) {
	return configPath(explicit)
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func SaveConfig(path string, cfg Config) error {
	if path == "" {
		paths, err := Paths()
		if err != nil {
			return err
		}
		path = paths.ConfigPath
	}

	if err := ensureParentDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
