package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunPauseStopNextPrev(t *testing.T) {
	t.Parallel()

	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	for _, cmd := range []string{"pause", "stop", "next", "prev"} {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", cmd}, &out, &errOut)
		if code != 0 {
			t.Fatalf("%s exit code = %d; stderr=%q", cmd, code, errOut.String())
		}
	}

	want := []string{"/Pause", "/Stop", "/Skip", "/Back"}
	for i := range want {
		if gotURL := <-got; gotURL != want[i] {
			t.Fatalf("call[%d] = %q; want %q", i, gotURL, want[i])
		}
	}
}

func TestRunShuffleAndRepeat(t *testing.T) {
	t.Parallel()

	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	cases := [][]string{
		{"shuffle", "on"},
		{"shuffle", "off"},
		{"repeat", "track"},
		{"repeat", "queue"},
		{"repeat", "off"},
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
		"/Shuffle?state=1",
		"/Shuffle?state=0",
		"/Repeat?state=1",
		"/Repeat?state=0",
		"/Repeat?state=2",
	}
	for i := range want {
		if gotURL := <-got; gotURL != want[i] {
			t.Fatalf("call[%d] = %q; want %q", i, gotURL, want[i])
		}
	}
}

func TestRunMuteToggle(t *testing.T) {
	t.Parallel()

	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		switch r.URL.Path {
		case "/Status":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<status mute="1" volume="10" state="play" artist="A" title1="T"/>`))
		default:
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ok/>`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "mute", "toggle"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}

	if gotURL := <-got; gotURL != "/Status" {
		t.Fatalf("call[0] = %q; want /Status", gotURL)
	}
	if gotURL := <-got; !strings.HasPrefix(gotURL, "/Volume?") || !strings.Contains(gotURL, "mute=0") || !strings.Contains(gotURL, "tell_slaves=1") {
		t.Fatalf("call[1] = %q; want /Volume with mute=0 tell_slaves=1", gotURL)
	}
}
