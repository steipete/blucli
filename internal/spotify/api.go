package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIOptions struct {
	APIBaseURL string
	HTTP       *http.Client
}

type API struct {
	apiBaseURL string
	http       *http.Client
}

func NewAPI(opts APIOptions) *API {
	base := strings.TrimSpace(opts.APIBaseURL)
	if base == "" {
		base = "https://api.spotify.com"
	}
	httpClient := opts.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &API{
		apiBaseURL: base,
		http:       httpClient,
	}
}

type Device struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	IsActive      bool   `json:"is_active"`
	VolumePercent int    `json:"volume_percent"`
}

type DevicesResponse struct {
	Devices []Device `json:"devices"`
}

func (a *API) Devices(ctx context.Context, accessToken string) (DevicesResponse, error) {
	var dr DevicesResponse
	if err := a.doJSON(ctx, accessToken, http.MethodGet, "/v1/me/player/devices", nil, &dr); err != nil {
		return DevicesResponse{}, err
	}
	return dr, nil
}

type SearchTrack struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

type SearchArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type SearchResponse struct {
	Tracks struct {
		Items []SearchTrack `json:"items"`
	} `json:"tracks"`
	Artists struct {
		Items []SearchArtist `json:"items"`
	} `json:"artists"`
}

func (a *API) Search(ctx context.Context, accessToken, query string, types []string, limit int) (SearchResponse, error) {
	q := url.Values{}
	q.Set("q", query)
	if len(types) == 0 {
		types = []string{"track", "artist"}
	}
	q.Set("type", strings.Join(types, ","))
	if limit <= 0 {
		limit = 5
	}
	q.Set("limit", fmt.Sprintf("%d", limit))

	var sr SearchResponse
	if err := a.doJSON(ctx, accessToken, http.MethodGet, "/v1/search?"+q.Encode(), nil, &sr); err != nil {
		return SearchResponse{}, err
	}
	return sr, nil
}

type ArtistTopTracksResponse struct {
	Tracks []struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	} `json:"tracks"`
}

func (a *API) ArtistTopTracks(ctx context.Context, accessToken, artistID, market string) (ArtistTopTracksResponse, error) {
	if strings.TrimSpace(market) == "" {
		market = "US"
	}
	path := fmt.Sprintf("/v1/artists/%s/top-tracks?market=%s", url.PathEscape(artistID), url.QueryEscape(market))
	var resp ArtistTopTracksResponse
	if err := a.doJSON(ctx, accessToken, http.MethodGet, path, nil, &resp); err != nil {
		return ArtistTopTracksResponse{}, err
	}
	return resp, nil
}

type TransferRequest struct {
	DeviceIDs []string `json:"device_ids"`
	Play      bool     `json:"play"`
}

func (a *API) Transfer(ctx context.Context, accessToken, deviceID string, play bool) error {
	if strings.TrimSpace(deviceID) == "" {
		return errors.New("missing device id")
	}
	body := TransferRequest{DeviceIDs: []string{deviceID}, Play: play}
	return a.doNoContent(ctx, accessToken, http.MethodPut, "/v1/me/player", body)
}

type PlayRequest struct {
	URIs       []string `json:"uris,omitempty"`
	ContextURI string   `json:"context_uri,omitempty"`
}

func (a *API) Play(ctx context.Context, accessToken, deviceID string, body PlayRequest) error {
	if strings.TrimSpace(deviceID) == "" {
		return errors.New("missing device id")
	}
	path := "/v1/me/player/play?device_id=" + url.QueryEscape(deviceID)
	return a.doNoContent(ctx, accessToken, http.MethodPut, path, body)
}

func (a *API) doNoContent(ctx context.Context, accessToken, method, path string, body any) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.apiBaseURL+path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("spotify: unexpected status: %s", resp.Status)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (a *API) doJSON(ctx context.Context, accessToken, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, a.apiBaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("spotify: unexpected status: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func TokenExpired(tok Token, skew time.Duration) bool {
	if tok.AccessToken == "" {
		return true
	}
	if tok.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(tok.ExpiresAt.Add(-skew))
}
