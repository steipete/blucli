package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/spotify"
)

func TestRunSpotifyOpen(t *testing.T) {
	t.Parallel()

	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "spotify", "open"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if u := <-got; u != "/Play?url=Spotify%3Aplay" {
		t.Fatalf("url = %q; want /Play?url=Spotify%%3Aplay", u)
	}
}

func TestRunSpotifyLoginMissingClientID(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "")

	cfgPath := writeTestConfig(t, "http://127.0.0.1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "spotify", "login", "--no-open"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit code = %d; want 2", code)
	}
	if got := errOut.String(); !strings.Contains(got, "missing client id") {
		t.Fatalf("stderr = %q; want missing client id", got)
	}
}

func TestRunSpotifyDevicesMissingClientID(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "")

	cfgPath := writeTestConfig(t, "http://127.0.0.1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "spotify", "devices"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit code = %d; want 1", code)
	}
	if got := errOut.String(); !strings.Contains(got, "missing spotify client id") {
		t.Fatalf("stderr = %q; want missing spotify client id", got)
	}
}

func TestRunSpotifySearchMissingQuery(t *testing.T) {
	t.Parallel()

	cfgPath := writeTestConfig(t, "http://127.0.0.1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "spotify", "search"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit code = %d; want 2", code)
	}
	if got := errOut.String(); !strings.Contains(got, "missing query") {
		t.Fatalf("stderr = %q; want missing query", got)
	}
}

func TestRunSpotifyPlayMissingQuery(t *testing.T) {
	t.Parallel()

	cfgPath := writeTestConfig(t, "http://127.0.0.1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "spotify", "play"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit code = %d; want 2", code)
	}
	if got := errOut.String(); !strings.Contains(got, "missing query") {
		t.Fatalf("stderr = %q; want missing query", got)
	}
}

func TestRunSpotifyLogoutWritesConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := []byte(`{
  "default_device": "http://127.0.0.1",
  "spotify": {
    "client_id": "cid",
    "token": {
      "access_token": "at",
      "refresh_token": "rt",
      "expires_at": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339Nano) + `"
    }
  }
}
`)
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "spotify", "logout"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}

	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(after), "access_token") {
		t.Fatalf("config still contains access_token: %s", string(after))
	}
}

func TestSpotifyHelpers(t *testing.T) {
	t.Parallel()

	dev, ok := matchSpotifyDevice([]spotify.Device{
		{ID: "1", Name: "Office", IsActive: false},
		{ID: "2", Name: "Kitchen", IsActive: true},
	}, "office")
	if !ok || dev.ID != "1" {
		t.Fatalf("match exact: ok=%v dev=%+v; want id=1", ok, dev)
	}

	dev, ok = matchSpotifyDevice([]spotify.Device{
		{ID: "1", Name: "Room One", IsActive: false},
		{ID: "2", Name: "Other", IsActive: false},
	}, "room")
	if !ok || dev.ID != "1" {
		t.Fatalf("match contains: ok=%v dev=%+v; want id=1", ok, dev)
	}

	dev, ok = matchSpotifyDevice([]spotify.Device{
		{ID: "2", Name: "Other", IsActive: true},
		{ID: "1", Name: "Room", IsActive: false},
	}, "")
	if !ok || dev.ID != "2" {
		t.Fatalf("match active: ok=%v dev=%+v; want id=2", ok, dev)
	}

	dev, ok = matchSpotifyDevice([]spotify.Device{{ID: "1", Name: "Only"}}, "")
	if !ok || dev.ID != "1" {
		t.Fatalf("match single: ok=%v dev=%+v; want id=1", ok, dev)
	}

	res := spotify.SearchResponse{}
	res.Artists.Items = []spotify.SearchArtist{{Name: "Garrett Emery"}}
	res.Tracks.Items = []spotify.SearchTrack{{URI: "spotify:track:1"}}
	if got := autoPickType("Garrett Emery", res); got != "artist" {
		t.Fatalf("autoPickType = %q; want artist", got)
	}
	res = spotify.SearchResponse{}
	res.Tracks.Items = []spotify.SearchTrack{{URI: "spotify:track:1"}}
	if got := autoPickType("anything", res); got != "track" {
		t.Fatalf("autoPickType = %q; want track", got)
	}

	if got := normalizeName("  Garrett   Emery  "); got != "garrett  emery" {
		t.Fatalf("normalizeName = %q; want garrett  emery", got)
	}
	if got := max0(-1); got != 0 {
		t.Fatalf("max0 = %d; want 0", got)
	}
}

func TestSpotifyAccessTokenErrorsAndFastPath(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "")

	p := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}

	if _, _, err := spotifyAccessToken(context.Background(), p, config.Config{}); err == nil {
		t.Fatalf("want error for missing client id")
	}

	if _, _, err := spotifyAccessToken(context.Background(), p, config.Config{Spotify: config.SpotifyConfig{ClientID: "cid"}}); err == nil {
		t.Fatalf("want error for missing token")
	}

	cfg := config.Config{
		Spotify: config.SpotifyConfig{
			ClientID: "cid",
			Token: config.SpotifyToken{
				AccessToken:  "at",
				RefreshToken: "",
				ExpiresAt:    time.Now().Add(-2 * time.Hour),
			},
		},
	}
	if _, _, err := spotifyAccessToken(context.Background(), p, cfg); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("err = %v; want expired token error", err)
	}

	cfg.Spotify.Token.ExpiresAt = time.Now().Add(2 * time.Hour)
	tok, _, err := spotifyAccessToken(context.Background(), p, cfg)
	if err != nil || tok != "at" {
		t.Fatalf("tok=%q err=%v; want at nil", tok, err)
	}
}
