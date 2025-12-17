package spotify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestOAuthRefresh(t *testing.T) {
	t.Parallel()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s; want POST", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","token_type":"Bearer","scope":"s1 s2","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)

	oauth, err := NewOAuth(OAuthOptions{AccountsBaseURL: srv.URL, ClientID: "CID", HTTP: srv.Client()})
	if err != nil {
		t.Fatalf("NewOAuth: %v", err)
	}

	tok, err := oauth.Refresh(context.Background(), "RT")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken != "AT" {
		t.Fatalf("access token = %q; want AT", tok.AccessToken)
	}
	if tok.RefreshToken != "" {
		t.Fatalf("refresh token = %q; want empty (often omitted on refresh)", tok.RefreshToken)
	}
	if tok.TokenType != "Bearer" {
		t.Fatalf("token type = %q; want Bearer", tok.TokenType)
	}
	if tok.Scope != "s1 s2" {
		t.Fatalf("scope = %q; want s1 s2", tok.Scope)
	}
	if tok.ExpiresAt.Before(time.Now().Add(30 * time.Minute)) {
		t.Fatalf("expiresAt = %v; want ~1h in future", tok.ExpiresAt)
	}

	v, _ := url.ParseQuery(gotBody)
	if got := v.Get("grant_type"); got != "refresh_token" {
		t.Fatalf("grant_type = %q; want refresh_token", got)
	}
	if got := v.Get("refresh_token"); got != "RT" {
		t.Fatalf("refresh_token = %q; want RT", got)
	}
	if got := v.Get("client_id"); got != "CID" {
		t.Fatalf("client_id = %q; want CID", got)
	}
}

func TestTokenExpired(t *testing.T) {
	t.Parallel()

	if !TokenExpired(Token{}, 0) {
		t.Fatalf("empty token should be expired")
	}
	if TokenExpired(Token{AccessToken: "AT"}, 0) {
		t.Fatalf("no expiresAt should be treated as not expired")
	}
	if !TokenExpired(Token{AccessToken: "AT", ExpiresAt: time.Now().Add(-time.Minute)}, 0) {
		t.Fatalf("past token should be expired")
	}
	if TokenExpired(Token{AccessToken: "AT", ExpiresAt: time.Now().Add(2 * time.Minute)}, time.Minute) {
		t.Fatalf("future token should not be expired with skew")
	}
}
