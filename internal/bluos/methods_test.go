package bluos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncStatusRequestAndParsing(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/SyncStatus" {
			t.Fatalf("path = %q; want /SyncStatus", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<SyncStatus group="G" volume="10" mute="0" etag="x"><master port="11000"> host </master><slave id="s1" port="11001"/></SyncStatus>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	s, err := client.SyncStatus(context.Background(), SyncStatusOptions{TimeoutSeconds: 30, ETag: "prev"})
	if err != nil {
		t.Fatalf("SyncStatus() err = %v", err)
	}
	if gotQuery != "etag=prev&timeout=30" && gotQuery != "timeout=30&etag=prev" {
		t.Fatalf("query = %q; want contains timeout and etag", gotQuery)
	}
	if s.Group != "G" || s.ETag != "x" || s.Master == nil || s.Master.Port != 11000 || s.Master.Host != "host" {
		t.Fatalf("sync = %+v", s)
	}
	if len(s.Slaves) != 1 || s.Slaves[0].ID != "s1" {
		t.Fatalf("slaves = %+v", s.Slaves)
	}
}

func TestPlaybackRequests(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.Play(context.Background(), PlayOptions{SeekSeconds: 7, ID: 2, URL: "http://x"}); err != nil {
		t.Fatalf("Play() err = %v", err)
	}
	if err := client.Pause(context.Background(), PauseOptions{Toggle: true}); err != nil {
		t.Fatalf("Pause() err = %v", err)
	}
	if err := client.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() err = %v", err)
	}
	if err := client.Skip(context.Background()); err != nil {
		t.Fatalf("Skip() err = %v", err)
	}
	if err := client.Back(context.Background()); err != nil {
		t.Fatalf("Back() err = %v", err)
	}
	if err := client.Shuffle(context.Background(), true); err != nil {
		t.Fatalf("Shuffle(true) err = %v", err)
	}
	if err := client.Shuffle(context.Background(), false); err != nil {
		t.Fatalf("Shuffle(false) err = %v", err)
	}

	want := []string{
		"/Play?id=2&seek=7&url=http%3A%2F%2Fx",
		"/Pause?toggle=1",
		"/Stop",
		"/Skip",
		"/Back",
		"/Shuffle?state=1",
		"/Shuffle?state=0",
	}
	for i := range want {
		if got := <-seen; got != want[i] {
			t.Fatalf("call[%d] = %q; want %q", i, got, want[i])
		}
	}
}

func TestRepeatOutOfRange(t *testing.T) {
	t.Parallel()

	baseURL, _ := url.Parse("http://127.0.0.1")
	client := NewClient(baseURL, Options{})

	if err := client.Repeat(context.Background(), 99); err == nil {
		t.Fatalf("want error")
	}
}

func TestVolumeDeltaDBAndMuteRequests(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.VolumeDeltaDB(context.Background(), VolumeDeltaDBOptions{DeltaDB: 0, TellSlaves: true}); err != nil {
		t.Fatalf("VolumeDeltaDB(0) err = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("calls = %d; want 0", calls.Load())
	}

	if err := client.VolumeDeltaDB(context.Background(), VolumeDeltaDBOptions{DeltaDB: -3, TellSlaves: true}); err != nil {
		t.Fatalf("VolumeDeltaDB(-3) err = %v", err)
	}
	if err := client.VolumeMute(context.Background(), VolumeMuteOptions{Mute: true, TellSlaves: true}); err != nil {
		t.Fatalf("VolumeMute(true) err = %v", err)
	}
	if err := client.VolumeMute(context.Background(), VolumeMuteOptions{Mute: false, TellSlaves: false}); err != nil {
		t.Fatalf("VolumeMute(false) err = %v", err)
	}

	want := []string{
		"/Volume?db=-3&tell_slaves=1",
		"/Volume?mute=1&tell_slaves=1",
		"/Volume?mute=0",
	}
	for i := range want {
		if gotURL := <-got; gotURL != want[i] {
			t.Fatalf("call[%d] = %q; want %q", i, gotURL, want[i])
		}
	}
}

func TestRemoveSlaveRequestAndDefaults(t *testing.T) {
	t.Parallel()

	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if err := client.RemoveSlave(context.Background(), RemoveSlaveOptions{SlaveHost: "192.0.2.1"}); err != nil {
		t.Fatalf("RemoveSlave() err = %v", err)
	}
	if u := <-got; u != "/RemoveSlave?port=11000&slave=192.0.2.1" {
		t.Fatalf("url = %q; want /RemoveSlave?port=11000&slave=192.0.2.1", u)
	}
}

func TestQueueMutationsAndSave(t *testing.T) {
	t.Parallel()

	got := make(chan string, 8)
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

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if _, err := client.Clear(context.Background()); err != nil {
		t.Fatalf("Clear() err = %v", err)
	}
	if _, err := client.Delete(context.Background(), 7); err != nil {
		t.Fatalf("Delete() err = %v", err)
	}
	if _, err := client.Move(context.Background(), 1, 2); err != nil {
		t.Fatalf("Move() err = %v", err)
	}
	if _, err := client.Save(context.Background(), "My List"); err != nil {
		t.Fatalf("Save() err = %v", err)
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

func TestQueueErrors(t *testing.T) {
	t.Parallel()

	baseURL, _ := url.Parse("http://127.0.0.1")
	client := NewClient(baseURL, Options{})

	if _, err := client.Delete(context.Background(), -1); err == nil {
		t.Fatalf("want error")
	}
	if _, err := client.Move(context.Background(), -1, 2); err == nil {
		t.Fatalf("want error")
	}
	if _, err := client.Save(context.Background(), ""); err == nil {
		t.Fatalf("want error")
	}
}

func TestLoadPresetFallbacks(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Preset" {
			t.Fatalf("path = %q; want /Preset", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Query().Get("id") {
		case "loaded":
			_, _ = w.Write([]byte(`<loaded service="Spotify" entries="1"/>`))
		case "presets":
			_, _ = w.Write([]byte(`<presets><preset id="1" name="A"/></presets>`))
		default:
			_, _ = w.Write([]byte(`<weird/>`))
		}
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if _, err := client.LoadPreset(context.Background(), ""); err == nil {
		t.Fatalf("want error for missing id")
	}

	v, err := client.LoadPreset(context.Background(), "loaded")
	if err != nil {
		t.Fatalf("LoadPreset(loaded) err = %v", err)
	}
	if _, ok := v.(LoadedResponse); !ok {
		t.Fatalf("type = %T; want LoadedResponse", v)
	}

	v, err = client.LoadPreset(context.Background(), "presets")
	if err != nil {
		t.Fatalf("LoadPreset(presets) err = %v", err)
	}
	if _, ok := v.(Presets); !ok {
		t.Fatalf("type = %T; want Presets", v)
	}

	v, err = client.LoadPreset(context.Background(), "weird")
	if err != nil {
		t.Fatalf("LoadPreset(weird) err = %v", err)
	}
	m, ok := v.(map[string]string)
	if !ok || m["xml"] == "" {
		t.Fatalf("type=%T value=%v; want xml map", v, v)
	}
}

func TestPlaylistsRadioBrowseSleepAndRawGet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/Playlists":
			if r.URL.Query().Get("service") != "Spotify" || r.URL.Query().Get("category") != "playlists" || r.URL.Query().Get("expr") != "q" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`<playlists><name>A</name></playlists>`))
		case "/RadioBrowse":
			if r.URL.Query().Get("service") != "TuneIn" || r.URL.Query().Get("expr") != "Garrett" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`<radiotime service="TuneIn"><item id="i1" text="X"/></radiotime>`))
		case "/Sleep":
			_, _ = w.Write([]byte("15"))
		case "/Status":
			if r.URL.Query().Get("foo") != "bar" {
				t.Fatalf("raw query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`<status/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	pl, err := client.Playlists(context.Background(), PlaylistsOptions{Service: "Spotify", Category: "playlists", Expr: "q"})
	if err != nil {
		t.Fatalf("Playlists() err = %v", err)
	}
	if len(pl.Names) != 1 || strings.TrimSpace(pl.Names[0].Text) != "A" {
		t.Fatalf("playlists = %+v", pl)
	}

	rb, err := client.RadioBrowse(context.Background(), RadioBrowseOptions{Service: "TuneIn", Expr: "Garrett"})
	if err != nil {
		t.Fatalf("RadioBrowse() err = %v", err)
	}
	if len(rb.Items) != 1 || rb.Items[0].ID != "i1" {
		t.Fatalf("radiobrowse = %+v", rb)
	}

	mins, err := client.Sleep(context.Background())
	if err != nil || mins != 15 {
		t.Fatalf("Sleep() mins=%d err=%v; want 15 nil", mins, err)
	}

	data, err := client.RawGet(context.Background(), "/Status", map[string]string{"foo": "bar"}, false)
	if err != nil {
		t.Fatalf("RawGet() err = %v", err)
	}
	if !strings.Contains(string(data), "<status") {
		t.Fatalf("raw = %q; want <status", string(data))
	}
}

func TestAddRemoveSlaveMissingHost(t *testing.T) {
	t.Parallel()

	baseURL, _ := url.Parse("http://127.0.0.1")
	client := NewClient(baseURL, Options{})

	if err := client.AddSlave(context.Background(), AddSlaveOptions{}); err == nil {
		t.Fatalf("want error")
	}
	if err := client.RemoveSlave(context.Background(), RemoveSlaveOptions{}); err == nil {
		t.Fatalf("want error")
	}
}

func TestHTTPNon2xxErrorIncludesBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	_, err := client.Status(context.Background(), StatusOptions{})
	if err == nil || !strings.Contains(err.Error(), "http 418") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("err = %v; want http status+body", err)
	}
}

func TestSleepBlankIsZero(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Sleep" {
			t.Fatalf("path = %q; want /Sleep", r.URL.Path)
		}
		_, _ = w.Write([]byte(" \n "))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	mins, err := client.Sleep(context.Background())
	if err != nil || mins != 0 {
		t.Fatalf("mins=%d err=%v; want 0 nil", mins, err)
	}
}

func TestRawGetMutatingUsesWrite(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		if r.URL.Path != "/Play" {
			t.Fatalf("path = %q; want /Play", r.URL.Path)
		}
		if r.URL.Query().Get("url") != "x" {
			t.Fatalf("query = %q; want url=x", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`<ok/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	if _, err := client.RawGet(context.Background(), "/Play", map[string]string{"url": "x"}, true); err != nil {
		t.Fatalf("RawGet(mutating) err = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d; want 1", calls.Load())
	}
}

func TestTraceWritesRequestLine(t *testing.T) {
	t.Parallel()

	var trace strings.Builder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status/>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{Trace: &trace})

	if _, err := client.Status(context.Background(), StatusOptions{}); err != nil {
		t.Fatalf("Status() err = %v", err)
	}
	if got := trace.String(); got == "" || !strings.Contains(got, "http: GET") || !strings.Contains(got, "/Status") {
		t.Fatalf("trace = %q; want request line", got)
	}
}

func TestContextCancellationReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(1 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Status(ctx, StatusOptions{})
	if err == nil {
		t.Fatalf("want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; want context.Canceled", err)
	}
}
