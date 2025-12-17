package spotify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPISearch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer AT" {
			t.Fatalf("auth = %q; want Bearer AT", got)
		}
		if got := r.URL.Query().Get("q"); got != "garrett" {
			t.Fatalf("q = %q; want garrett", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tracks": {"items": [{"name":"T","uri":"spotify:track:1","artists":[{"name":"A"}]}]},
			"artists": {"items": [{"id":"ar1","name":"Garrett","uri":"spotify:artist:ar1"}]}
		}`))
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	res, err := api.Search(context.Background(), "AT", "garrett", []string{"track", "artist"}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Tracks.Items) != 1 || strings.TrimSpace(res.Tracks.Items[0].URI) != "spotify:track:1" {
		t.Fatalf("tracks = %+v; want 1 item with uri spotify:track:1", res.Tracks.Items)
	}
	if len(res.Artists.Items) != 1 || strings.TrimSpace(res.Artists.Items[0].ID) != "ar1" {
		t.Fatalf("artists = %+v; want 1 item with id ar1", res.Artists.Items)
	}
}

func TestAPIPlay(t *testing.T) {
	t.Parallel()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/me/player/play") {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("device_id"); got != "dev1" {
			t.Fatalf("device_id = %q; want dev1", got)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	if err := api.Play(context.Background(), "AT", "dev1", PlayRequest{URIs: []string{"spotify:track:1"}}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if gotBody == "" || !strings.Contains(gotBody, "spotify:track:1") {
		t.Fatalf("body = %q; want contains spotify:track:1", gotBody)
	}
}
