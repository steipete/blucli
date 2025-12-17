package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunBrowse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Browse" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("key"); got != "TuneIn:" {
			t.Fatalf("key = %q; want TuneIn:", got)
		}
		if got := r.URL.Query().Get("q"); got != "abc" {
			t.Fatalf("q = %q; want abc", got)
		}
		if got := r.URL.Query().Get("withContextMenuItems"); got != "1" {
			t.Fatalf("withContextMenuItems = %q; want 1", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<browse><item text="X" browseKey="K"/></browse>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "browse", "--key", "TuneIn:", "--q", "abc", "--context"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "X") {
		t.Fatalf("stdout = %q; want contains X", got)
	}
}

func TestRunPlaylists(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Playlists" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("service"); got != "Spotify" {
			t.Fatalf("service = %q; want Spotify", got)
		}
		if got := r.URL.Query().Get("category"); got != "artists" {
			t.Fatalf("category = %q; want artists", got)
		}
		if got := r.URL.Query().Get("expr"); got != "Garrett" {
			t.Fatalf("expr = %q; want Garrett", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<playlists><name>One</name><name>Two</name></playlists>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "playlists", "--service", "Spotify", "--category", "artists", "--expr", "Garrett"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "One") || !strings.Contains(got, "Two") {
		t.Fatalf("stdout = %q; want contains One and Two", got)
	}
}

func TestRunInputsListAndPlay(t *testing.T) {
	t.Parallel()

	var playCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/RadioBrowse":
			if got := r.URL.Query().Get("service"); got != "Capture" {
				t.Fatalf("service = %q; want Capture", got)
			}
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<radiotime service="Capture"><item id="Spotify" text="Spotify" URL="Spotify%3Aplay" type="audio"/></radiotime>`))
		case "/Play":
			playCalls++
			if got := r.URL.Query().Get("url"); got != "Spotify:play" {
				t.Fatalf("url = %q; want Spotify:play", got)
			}
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ok/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "inputs"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
		}
		if got := out.String(); !strings.Contains(got, "Spotify") {
			t.Fatalf("stdout = %q; want contains Spotify", got)
		}
	}
	{
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "inputs", "play", "Spotify"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
		}
	}
	if playCalls != 1 {
		t.Fatalf("play calls = %d; want 1", playCalls)
	}
}
