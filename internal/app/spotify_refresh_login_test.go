package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func TestSpotifyAccessToken_RefreshPath_StubbedAccounts(t *testing.T) {
	var gotBody string
	accountsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT2","refresh_token":"","token_type":"Bearer","scope":"s","expires_in":3600}`))
	}))
	t.Cleanup(accountsSrv.Close)
	t.Setenv("BLU_SPOTIFY_ACCOUNTS_BASE_URL", accountsSrv.URL)

	p := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}
	cfg := config.Config{
		Spotify: config.SpotifyConfig{
			ClientID: "CID",
			Token: config.SpotifyToken{
				AccessToken:  "AT",
				RefreshToken: "RT",
				ExpiresAt:    time.Now().Add(-2 * time.Hour),
				TokenType:    "Bearer",
			},
		},
	}

	accessToken, newCfg, err := spotifyAccessToken(context.Background(), p, cfg)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if accessToken != "AT2" {
		t.Fatalf("accessToken = %q; want AT2", accessToken)
	}
	if newCfg.Spotify.Token.AccessToken != "AT2" {
		t.Fatalf("cfg accessToken = %q; want AT2", newCfg.Spotify.Token.AccessToken)
	}
	if !strings.Contains(gotBody, "grant_type=refresh_token") {
		t.Fatalf("body = %q; want refresh_token grant", gotBody)
	}

	after, err := os.ReadFile(p.ConfigPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !bytes.Contains(after, []byte(`"access_token": "AT2"`)) {
		t.Fatalf("config = %s; want updated token", string(after))
	}
}

func TestSpotifyLogin_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}
	cfg := config.Config{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
	code := cmdSpotifyLogin(ctx, out, p, cfg, []string{"--client-id", "CID", "--redirect", "http://127.0.0.1:0/callback", "--no-open"})
	if code != 1 {
		t.Fatalf("code=%d; want 1", code)
	}
	if got := stderr.String(); !strings.Contains(got, "cancelled") {
		t.Fatalf("stderr = %q; want cancelled", got)
	}
}
