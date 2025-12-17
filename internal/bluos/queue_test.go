package bluos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPlaylistParsing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Playlist" {
			t.Fatalf("path = %q; want /Playlist", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<playlist name="Q" modified="0" length="2" id="44">
<song id="0" service="Tidal"><title>T1</title><art>A1</art><alb>B1</alb><fn>F1</fn></song>
<song id="1" service="Tidal"><title>T2</title><art>A2</art><alb>B2</alb><fn>F2</fn></song>
</playlist>`))
	}))
	t.Cleanup(srv.Close)

	baseURL, _ := url.Parse(srv.URL)
	client := NewClient(baseURL, Options{})

	pl, err := client.Playlist(context.Background(), PlaylistOptions{})
	if err != nil {
		t.Fatalf("Playlist() err = %v", err)
	}
	if pl.Length != 2 || len(pl.Songs) != 2 {
		t.Fatalf("len = %d songs=%d; want 2/2", pl.Length, len(pl.Songs))
	}
	if pl.Songs[0].Title != "T1" || pl.Songs[1].Title != "T2" {
		t.Fatalf("titles = %q,%q", pl.Songs[0].Title, pl.Songs[1].Title)
	}
}
