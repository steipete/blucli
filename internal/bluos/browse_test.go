package bluos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBrowseParsing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Browse" {
			t.Fatalf("path = %q; want /Browse", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<browse sid="5" type="menu">
<item text="TuneIn" browseKey="TuneIn:" type="link"/>
<item text="Spotify" playURL="/Play?url=Spotify%3Aplay" type="audio"/>
</browse>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	b, err := client.Browse(context.Background(), BrowseOptions{Key: "root"})
	if err != nil {
		t.Fatalf("Browse() err = %v", err)
	}
	if len(b.Items) != 2 {
		t.Fatalf("items = %d; want 2", len(b.Items))
	}
	if b.Items[0].BrowseKey != "TuneIn:" {
		t.Fatalf("browseKey = %q", b.Items[0].BrowseKey)
	}
	if b.Items[1].PlayURL == "" {
		t.Fatalf("playURL empty")
	}
}

func TestPresetsParsing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Presets" {
			t.Fatalf("path = %q; want /Presets", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<presets prid="0"><preset id="1" name="Dad" url="/Load?x=1"/></presets>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	p, err := client.Presets(context.Background())
	if err != nil {
		t.Fatalf("Presets() err = %v", err)
	}
	if len(p.Presets) != 1 || p.Presets[0].ID != 1 {
		t.Fatalf("presets = %+v", p.Presets)
	}
}
