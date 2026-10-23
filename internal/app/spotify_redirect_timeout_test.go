package app

import (
	"bytes"
	"context"
	"io"
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

func TestSpotifyRequestsTimeOut(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		expired bool
	}{
		{name: "devices", args: []string{"spotify", "devices"}},
		{name: "search", args: []string{"spotify", "search", "synthetic"}},
		{name: "play", args: []string{"spotify", "play", "--no-activate", "--spotify-device", "synthetic", "synthetic"}},
		{name: "refresh", args: []string{"spotify", "devices"}, expired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				select {
				case <-r.Context().Done():
				case <-time.After(2 * time.Second):
				}
			}))
			defer srv.Close()
			t.Setenv("BLU_SPOTIFY_API_BASE_URL", srv.URL)
			t.Setenv("BLU_SPOTIFY_ACCOUNTS_BASE_URL", srv.URL)
			expiry := time.Now().Add(time.Hour)
			if tc.expired {
				expiry = time.Now().Add(-time.Hour)
			}
			cfg := config.Config{DefaultDevice: "127.0.0.1:11000", Spotify: config.SpotifyConfig{ClientID: "synthetic-client", Token: config.SpotifyToken{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh", ExpiresAt: expiry}}}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := config.SaveConfig(path, cfg); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			args := append([]string{"--config", path, "--timeout", "40ms"}, tc.args...)
			start := time.Now()
			code := Run(ctx, args, &stdout, &stderr)
			if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
				t.Fatalf("request ignored the configured timeout: %s", elapsed)
			}
			if code == 0 {
				t.Fatal("stalled request should fail")
			}
			if !strings.Contains(stderr.String(), "deadline exceeded") && !strings.Contains(stderr.String(), "Timeout") {
				t.Fatalf("expected a timeout, got %q", stderr.String())
			}
		})
	}
}
