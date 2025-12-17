package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunQueueOps(t *testing.T) {
	t.Parallel()

	got := make(chan string, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/Save":
			_, _ = w.Write([]byte(`<saved entries="3"/>`))
		default:
			_, _ = w.Write([]byte(`<playlist id="1" length="0"/>`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	cases := [][]string{
		{"queue", "clear"},
		{"queue", "delete", "7"},
		{"queue", "move", "1", "2"},
		{"queue", "save", "My", "List"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), append([]string{"--config", cfgPath, "--discover=false"}, args...), &out, &errOut)
		if code != 0 {
			t.Fatalf("%v exit code = %d; stderr=%q", args, code, errOut.String())
		}
	}

	want := []string{
		"/Clear",
		"/Delete?id=7",
		"/Move?new=2&old=1",
		"/Save?name=My+List",
	}
	for i := range want {
		if gotURL := <-got; gotURL != want[i] {
			t.Fatalf("call[%d] = %q; want %q", i, gotURL, want[i])
		}
	}
}

func TestRunPresetsListAndLoad(t *testing.T) {
	t.Parallel()

	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/Presets":
			_, _ = w.Write([]byte(`<presets><preset id="1" name="A"/></presets>`))
		case "/Preset":
			_, _ = w.Write([]byte(`<loaded service="Spotify" entries="1"/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "presets", "list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("list exit code = %d; stderr=%q", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "presets", "load", "2"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("load exit code = %d; stderr=%q", code, errOut.String())
	}

	if gotURL := <-got; gotURL != "/Presets" {
		t.Fatalf("call[0] = %q; want /Presets", gotURL)
	}
	if gotURL := <-got; gotURL != "/Preset?id=2" {
		t.Fatalf("call[1] = %q; want /Preset?id=2", gotURL)
	}
}
