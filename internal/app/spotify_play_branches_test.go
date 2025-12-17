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

func TestSpotifyPlay_Branches(t *testing.T) {
	var playCalls int
	spotifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			q := r.URL.Query().Get("q")
			switch q {
			case "track-missing-uri":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"T","uri":"","artists":[{"name":"A"}]}]},"artists":{"items":[]}}`))
			case "artist-top-empty":
				_, _ = w.Write([]byte(`{"tracks":{"items":[]},"artists":{"items":[{"id":"empty","name":"X","uri":"spotify:artist:empty"}]}}`))
			case "artist-top-nouri":
				_, _ = w.Write([]byte(`{"tracks":{"items":[]},"artists":{"items":[{"id":"nouri","name":"X","uri":"spotify:artist:nouri"}]}}`))
			default:
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"T","uri":"spotify:track:1","artists":[{"name":"A"}]}]},"artists":{"items":[{"id":"ar1","name":"A","uri":"spotify:artist:ar1"}]}}`))
			}
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/top-tracks"):
			if strings.Contains(r.URL.Path, "/empty/") {
				_, _ = w.Write([]byte(`{"tracks":[]}`))
				return
			}
			if strings.Contains(r.URL.Path, "/nouri/") {
				_, _ = w.Write([]byte(`{"tracks":[{"name":"t1","uri":""}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"tracks":[{"name":"t1","uri":"spotify:track:11"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/me/player":
			// Make Transfer() fail; cmdSpotifyPlay treats it as best-effort.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/me/player/play":
			playCalls++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(spotifySrv.Close)
	t.Setenv("BLU_SPOTIFY_API_BASE_URL", spotifySrv.URL)

	cfg := config.Config{
		DefaultDevice: "http://127.0.0.1:11000",
		Spotify: config.SpotifyConfig{
			ClientID: "CID",
			Token: config.SpotifyToken{
				AccessToken:  "AT",
				RefreshToken: "RT",
				ExpiresAt:    time.Now().Add(2 * time.Hour),
			},
		},
	}
	paths := config.PathSet{ConfigPath: filepath.Join(t.TempDir(), "config.json")}

	run := func(args []string) (int, string, string) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
		code := cmdSpotifyPlay(context.Background(), out, paths, cfg, config.DiscoveryCache{}, "", false, 0, 2*time.Second, false, nil, args)
		return code, stdout.String(), stderr.String()
	}

	if code, _, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "nope", "q"}); code != 2 || !strings.Contains(errOut, "unknown --type") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}

	if code, _, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "track", "--pick", "5", "q"}); code != 2 || !strings.Contains(errOut, "pick out of range") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}

	if code, _, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "track", "track-missing-uri"}); code != 1 || !strings.Contains(errOut, "missing track uri") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}

	if code, _, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "artist", "artist-top-empty"}); code != 1 || !strings.Contains(errOut, "artist has no top tracks") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}

	if code, _, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "artist", "artist-top-nouri"}); code != 1 || !strings.Contains(errOut, "missing uris") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}

	code, stdout, errOut := run([]string{"--no-activate", "--spotify-device", "dev1", "--type", "auto", "q"})
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	if !strings.Contains(stdout, "\"type\": \"track\"") {
		t.Fatalf("stdout=%q", stdout)
	}
	if playCalls == 0 {
		t.Fatalf("expected play calls")
	}
}
