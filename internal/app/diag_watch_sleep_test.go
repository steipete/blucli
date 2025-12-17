package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunDiag(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/Status":
			_, _ = w.Write([]byte(`<status state="play" volume="10" mute="0" artist="A" title1="T" etag="s1" name="Room" model="M"/>`))
		case "/SyncStatus":
			_, _ = w.Write([]byte(`<SyncStatus group="G" volume="10" mute="0" etag="x"><master port="11000">host</master></SyncStatus>`))
		case "/Presets":
			_, _ = w.Write([]byte(`<presets><preset id="1" name="A"/></presets>`))
		case "/Playlist":
			_, _ = w.Write([]byte(`<playlist id="1" length="0"/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "diag"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "\"device\"") || !strings.Contains(got, "\"status\"") || !strings.Contains(got, "\"sync\"") {
		t.Fatalf("stdout = %q; want json report", got)
	}
}

func TestRunSleep(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Sleep" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("15"))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "sleep"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "15" {
		t.Fatalf("stdout = %q; want 15", got)
	}
}

func TestRunWatchStatusCancels(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status state="play" volume="10" mute="0" artist="A" title1="T" etag="e1"/>`))
		once.Do(func() {
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		})
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(ctx, []string{"--config", cfgPath, "--discover=false", "watch", "status"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !strings.Contains(got, "vol=10") {
		t.Fatalf("stdout = %q; want printed status", got)
	}
}

func TestRunWatchSyncCancels(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/SyncStatus" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<SyncStatus group="G" volume="10" mute="0" etag="x"><master port="11000">host</master></SyncStatus>`))
		once.Do(func() {
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		})
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(ctx, []string{"--config", cfgPath, "--discover=false", "watch", "sync"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !strings.Contains(got, "group: G") {
		t.Fatalf("stdout = %q; want printed syncstatus", got)
	}
}
