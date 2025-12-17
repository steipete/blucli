package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenType    string
	Scope        string
}

type OAuthOptions struct {
	AccountsBaseURL string
	ClientID        string
	HTTP            *http.Client
}

type OAuth struct {
	accountsBaseURL string
	clientID        string
	http            *http.Client
}

func NewOAuth(opts OAuthOptions) (*OAuth, error) {
	if strings.TrimSpace(opts.ClientID) == "" {
		return nil, errors.New("missing client id")
	}
	base := strings.TrimSpace(opts.AccountsBaseURL)
	if base == "" {
		base = "https://accounts.spotify.com"
	}
	httpClient := opts.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OAuth{
		accountsBaseURL: base,
		clientID:        strings.TrimSpace(opts.ClientID),
		http:            httpClient,
	}, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (o *OAuth) ExchangeAuthorizationCode(ctx context.Context, code, redirectURI, codeVerifier string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", o.clientID)
	form.Set("code_verifier", codeVerifier)
	return o.tokenRequest(ctx, form)
}

func (o *OAuth) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", o.clientID)
	return o.tokenRequest(ctx, form)
}

func (o *OAuth) tokenRequest(ctx context.Context, form url.Values) (Token, error) {
	endpoint := o.accountsBaseURL + "/api/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.http.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Token{}, fmt.Errorf("token: unexpected status: %s", resp.Status)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return Token{}, err
	}

	tok := Token{
		AccessToken:  strings.TrimSpace(tr.AccessToken),
		RefreshToken: strings.TrimSpace(tr.RefreshToken),
		TokenType:    strings.TrimSpace(tr.TokenType),
		Scope:        strings.TrimSpace(tr.Scope),
	}
	if tr.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tok, nil
}
