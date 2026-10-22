package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func TestSpotifyLoginRejectsRedirectWithoutPath(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("spotify login panicked on a redirect with no path: %v", recovered)
		}
	}()

	paths := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
	code := cmdSpotifyLogin(context.Background(), out, paths, config.Config{}, defaultHTTPTimeout, []string{
		"--client-id", "CID",
		"--redirect", "http://127.0.0.1:8974",
		"--no-open",
	})
	if code != 2 {
		t.Fatalf("code=%d stderr=%q; want 2", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "path") {
		t.Fatalf("stderr = %q; want a path error", stderr.String())
	}
}

func TestSpotifyDevicesTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BLU_SPOTIFY_API_BASE_URL", srv.URL)

	cfg := config.Config{
		Spotify: config.SpotifyConfig{
			ClientID: "CID",
			Token: config.SpotifyToken{
				AccessToken:  "AT",
				RefreshToken: "RT",
				ExpiresAt:    time.Now().Add(time.Hour),
			},
		},
	}
	paths := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})

	start := time.Now()
	code := cmdSpotifyDevices(context.Background(), out, paths, cfg, 300*time.Millisecond, nil)
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("devices blocked for %s", elapsed)
	}
	if code == 0 {
		t.Fatal("expected a timeout error")
	}
}
