package spotify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAPIDevices(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me/player/devices" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer AT" {
			t.Fatalf("auth = %q; want Bearer AT", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"devices":[{"id":"dev1","name":"Room","is_active":true,"type":"speaker","volume_percent":77}]}`))
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	devs, err := api.Devices(context.Background(), "AT")
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	if len(devs.Devices) != 1 || devs.Devices[0].ID != "dev1" || !devs.Devices[0].IsActive {
		t.Fatalf("devs = %+v", devs)
	}
}

func TestAPITransfer(t *testing.T) {
	t.Parallel()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me/player" || r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	if err := api.Transfer(context.Background(), "AT", "dev1", true); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if gotBody == "" || !strings.Contains(gotBody, "\"dev1\"") || !strings.Contains(gotBody, "\"play\":true") {
		t.Fatalf("body=%q", gotBody)
	}
	if err := api.Transfer(context.Background(), "AT", "", false); err == nil {
		t.Fatalf("want error for missing device id")
	}
}

func TestAPIArtistTopTracks_DefaultMarket(t *testing.T) {
	t.Parallel()

	var gotMarket string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/artists/ar1/top-tracks") {
			http.NotFound(w, r)
			return
		}
		gotMarket = r.URL.Query().Get("market")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tracks":[{"name":"t1","uri":"spotify:track:1"}]}`))
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	resp, err := api.ArtistTopTracks(context.Background(), "AT", "ar1", "")
	if err != nil {
		t.Fatalf("ArtistTopTracks: %v", err)
	}
	if gotMarket != "US" {
		t.Fatalf("market=%q; want US", gotMarket)
	}
	if len(resp.Tracks) != 1 || strings.TrimSpace(resp.Tracks[0].URI) != "spotify:track:1" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestOAuthExchangeAuthorizationCode(t *testing.T) {
	t.Parallel()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","refresh_token":"RT","token_type":"Bearer","scope":"s","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)

	oauth, err := NewOAuth(OAuthOptions{AccountsBaseURL: srv.URL, ClientID: "CID", HTTP: srv.Client()})
	if err != nil {
		t.Fatalf("NewOAuth: %v", err)
	}

	tok, err := oauth.ExchangeAuthorizationCode(context.Background(), "CODE", "http://127.0.0.1/callback", "VERIFIER")
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" || tok.TokenType != "Bearer" || tok.Scope != "s" {
		t.Fatalf("tok=%+v", tok)
	}
	if tok.ExpiresAt.Before(time.Now().Add(30 * time.Minute)) {
		t.Fatalf("expiresAt=%v; want ~1h", tok.ExpiresAt)
	}

	v, _ := url.ParseQuery(gotBody)
	if v.Get("grant_type") != "authorization_code" || v.Get("code") != "CODE" || v.Get("client_id") != "CID" {
		t.Fatalf("form=%q", gotBody)
	}
}

func TestSearch_DefaultTypesAndLimit(t *testing.T) {
	t.Parallel()

	var gotType string
	var gotLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			http.NotFound(w, r)
			return
		}
		gotType = r.URL.Query().Get("type")
		gotLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tracks":{"items":[]},"artists":{"items":[]}}`))
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	if _, err := api.Search(context.Background(), "AT", "q", nil, 0); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotType != "track,artist" || gotLimit != "5" {
		t.Fatalf("type=%q limit=%q", gotType, gotLimit)
	}
}

func TestNewCodeVerifier(t *testing.T) {
	t.Parallel()

	v, err := NewCodeVerifier()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(v) != 43 {
		t.Fatalf("len=%d; want 43", len(v))
	}
	if strings.ContainsAny(v, "+/=") {
		t.Fatalf("not base64url: %q", v)
	}
}

func TestDoJSON_Non2xx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"err":"nope"}`))
	}))
	t.Cleanup(srv.Close)

	api := NewAPI(APIOptions{APIBaseURL: srv.URL, HTTP: srv.Client()})
	var out any
	err := api.doJSON(context.Background(), "AT", http.MethodGet, "/v1/search?q=x&type=track&limit=1", nil, &out)
	if err == nil {
		t.Fatalf("want error")
	}
}

func TestTransfer_JSONMarshal(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(TransferRequest{DeviceIDs: []string{"d"}, Play: true})
	if string(body) == "" {
		t.Fatalf("empty")
	}
}
