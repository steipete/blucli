package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Status":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<status state="play" volume="15" mute="0" artist="A" title1="T"/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "status"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("vol=15")) {
		t.Fatalf("stdout = %q; want contains vol=15", got)
	}
}

func TestRunVersionCommand(t *testing.T) {
	Version = "v0.0.0-test"

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "v0.0.0-test\n" {
		t.Fatalf("stdout = %q; want v0.0.0-test", got)
	}
}

func TestRunVersionFlag(t *testing.T) {
	Version = "v0.0.0-test"

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "v0.0.0-test\n" {
		t.Fatalf("stdout = %q; want v0.0.0-test", got)
	}
}

func TestRunHelpFlag(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("blu — BluOS CLI")) {
		t.Fatalf("stdout = %q; want usage", got)
	}
}

func TestRunHelpCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("Usage:")) {
		t.Fatalf("stdout = %q; want usage", got)
	}
}

func TestRunDevicesHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"devices", "--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("blu devices")) {
		t.Fatalf("stdout = %q; want devices usage", got)
	}
}

func TestRunPlay(t *testing.T) {
	t.Parallel()

	gotPath := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath <- r.URL.Path
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "play"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := <-gotPath; got != "/Play" {
		t.Fatalf("path = %q; want /Play", got)
	}
}

func TestRunPlayWithURL(t *testing.T) {
	t.Parallel()

	gotURL := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "play", "--url", "http://example.com/stream.mp3"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := <-gotURL; got != "/Play?url=http%3A%2F%2Fexample.com%2Fstream.mp3" {
		t.Fatalf("url = %q; want /Play?url=http%%3A%%2F%%2Fexample.com%%2Fstream.mp3", got)
	}
}

func TestRunStatusDryRunAllowsReads(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Status":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<status state="play" volume="15" mute="0" artist="A" title1="T"/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "--dry-run", "status"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("vol=15")) {
		t.Fatalf("stdout = %q; want contains vol=15", got)
	}
	if got := errOut.String(); got == "" || !bytes.Contains([]byte(got), []byte("http: GET")) {
		t.Fatalf("stderr = %q; want contains http: GET", got)
	}
}

func TestRunPlayDryRunBlocksWrites(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "--dry-run", "play"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("server calls = %d; want 0", got)
	}
	if got := errOut.String(); got == "" || !bytes.Contains([]byte(got), []byte("/Play")) {
		t.Fatalf("stderr = %q; want contains /Play", got)
	}
}

func TestRunVolumeSet(t *testing.T) {
	t.Parallel()

	gotQuery := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Volume" {
			http.NotFound(w, r)
			return
		}
		gotQuery <- r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<volume/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "volume", "set", "10"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := <-gotQuery; got != "level=10&tell_slaves=1" {
		t.Fatalf("query = %q; want level=10&tell_slaves=1", got)
	}
}

func TestRunGroupAddParsesNameFlagAfterSlave(t *testing.T) {
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
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "group", "add", "192.0.2.1:11000", "--name", "Kitchen"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if u := <-got; u != "/AddSlave?group=Kitchen&port=11000&slave=192.0.2.1" {
		t.Fatalf("url = %q; want /AddSlave?group=Kitchen&port=11000&slave=192.0.2.1", u)
	}
}

func TestRunQueueListParsesFlagsAfterSubcommand(t *testing.T) {
	t.Parallel()

	gotQuery := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Playlist" {
			http.NotFound(w, r)
			return
		}
		gotQuery <- r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<playlist id="1" length="0"/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "queue", "list", "--start", "2", "--end", "3"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := <-gotQuery; got != "end=3&start=2" {
		t.Fatalf("query = %q; want end=3&start=2", got)
	}
}

func TestRunRawParsesFlagsAfterPath(t *testing.T) {
	t.Parallel()

	gotQuery := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery <- r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status/>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "--json", "raw", "/Status", "--param", "foo=bar"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := <-gotQuery; got != "foo=bar" {
		t.Fatalf("query = %q; want foo=bar", got)
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte(`"path": "/Status"`)) {
		t.Fatalf("stdout = %q; want contains \"path\": \"/Status\"", got)
	}
}

func TestRunTuneInSearch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/RadioBrowse" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("service"); got != "TuneIn" {
			t.Fatalf("service = %q; want TuneIn", got)
		}
		if got := r.URL.Query().Get("expr"); got != "Garrett" {
			t.Fatalf("expr = %q; want Garrett", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<radiotime service="TuneIn"><category text="Stations"><item id="s1" text="X" type="audio" URL="TuneIn%3As1"/></category></radiotime>`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--discover=false", "tunein", "search", "Garrett"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "Stations:") {
		t.Fatalf("stdout = %q; want contains Stations:", got)
	}
}

func writeTestConfig(t *testing.T, deviceURL string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"default_device":"` + deviceURL + `"}` + "\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
