package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func TestSpotifyDevicesSearchAndPlay_StubbedAPI(t *testing.T) {
	var (
		mu          sync.Mutex
		gotPlayBody string
		gotTransfer int
	)

	spotifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer AT" {
			t.Fatalf("auth = %q; want Bearer AT", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/me/player/devices":
			_, _ = w.Write([]byte(`{"devices":[{"id":"dev1","name":"Office","is_active":false}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			_, _ = w.Write([]byte(`{
			  "tracks": {"items": [{"name":"T","uri":"spotify:track:1","artists":[{"name":"A"}]}]},
			  "artists": {"items": [{"id":"ar1","name":"Garrett Emery","uri":"spotify:artist:ar1"}]}
			}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/artists/") && strings.HasSuffix(r.URL.Path, "/top-tracks"):
			_, _ = w.Write([]byte(`{"tracks":[{"name":"t1","uri":"spotify:track:11"},{"name":"t2","uri":"spotify:track:22"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/me/player":
			mu.Lock()
			gotTransfer++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/v1/me/player/play":
			if r.URL.Query().Get("device_id") != "dev1" {
				t.Fatalf("device_id = %q; want dev1", r.URL.Query().Get("device_id"))
			}
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			gotPlayBody = string(b)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(spotifySrv.Close)
	t.Setenv("BLU_SPOTIFY_API_BASE_URL", spotifySrv.URL)

	bluosSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status name="Office" state="play" volume="10" mute="0"/>`))
	}))
	t.Cleanup(bluosSrv.Close)

	cfg := config.Config{
		DefaultDevice: bluosSrv.URL,
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

	{
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
		code := cmdSpotifyDevices(context.Background(), out, paths, cfg, nil)
		if code != 0 {
			t.Fatalf("devices code=%d stderr=%q", code, stderr.String())
		}
		if got := stdout.String(); !strings.Contains(got, "\"devices\"") || !strings.Contains(got, "\"id\": \"dev1\"") {
			t.Fatalf("stdout = %q; want devices json", got)
		}
	}

	{
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
		code := cmdSpotifySearch(context.Background(), out, paths, cfg, []string{"Garrett"})
		if code != 0 {
			t.Fatalf("search code=%d stderr=%q", code, stderr.String())
		}
		if got := stdout.String(); !strings.Contains(got, "spotify:track:1") || !strings.Contains(got, "Garrett Emery") {
			t.Fatalf("stdout = %q; want search json", got)
		}
	}

	{
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
		code := cmdSpotifyPlay(context.Background(), out, paths, cfg, config.DiscoveryCache{}, "", false, 0, 3*time.Second, false, nil,
			[]string{"--no-activate", "--type", "track", "--pick", "0", "--wait", "50ms", "anything"},
		)
		if code != 0 {
			t.Fatalf("play(track) code=%d stderr=%q", code, stderr.String())
		}
		mu.Lock()
		body := gotPlayBody
		transfer := gotTransfer
		mu.Unlock()
		if transfer == 0 {
			t.Fatalf("want transfer call")
		}
		if body == "" || !strings.Contains(body, "spotify:track:1") {
			t.Fatalf("play body = %q; want track uri", body)
		}
	}

	{
		mu.Lock()
		gotPlayBody = ""
		gotTransfer = 0
		mu.Unlock()

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
		code := cmdSpotifyPlay(context.Background(), out, paths, cfg, config.DiscoveryCache{}, "", false, 0, 3*time.Second, false, nil,
			[]string{"--no-activate", "--type", "artist", "--pick", "0", "--market", "US", "--wait", "50ms", "Garrett Emery"},
		)
		if code != 0 {
			t.Fatalf("play(artist) code=%d stderr=%q", code, stderr.String())
		}
		mu.Lock()
		body := gotPlayBody
		transfer := gotTransfer
		mu.Unlock()
		if transfer == 0 {
			t.Fatalf("want transfer call")
		}
		if body == "" || !strings.Contains(body, "spotify:track:11") {
			t.Fatalf("play body = %q; want top tracks", body)
		}
	}
}

func TestSpotifyPlay_NoDeviceCandidates(t *testing.T) {
	var devicesCalls int
	spotifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/me/player/devices":
			devicesCalls++
			_, _ = w.Write([]byte(`{"devices":[]}`))
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})
	code := cmdSpotifyPlay(context.Background(), out, paths, cfg, config.DiscoveryCache{}, "", false, 0, 3*time.Second, false, nil,
		[]string{"--no-activate", "--wait", "1ms", "anything"},
	)
	if code != 1 {
		t.Fatalf("code=%d; want 1", code)
	}
	if devicesCalls == 0 {
		t.Fatalf("expected Devices calls")
	}
	if got := stderr.String(); !strings.Contains(got, "unable to pick Spotify Connect device") {
		t.Fatalf("stderr = %q; want pick error", got)
	}
}
