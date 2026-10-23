package app

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
	"github.com/steipete/blucli/internal/spotify"
)

func spotifyHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &http.Client{Timeout: timeout}
}

func cmdSpotifyDevices(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, httpTimeout time.Duration, args []string) int {
	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg, httpTimeout)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{
		APIBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_API_BASE_URL")),
		HTTP:       spotifyHTTPClient(httpTimeout),
	})
	devs, err := api.Devices(ctx, accessToken)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}
	out.Print(devs)
	return 0
}

func cmdSpotifySearch(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, httpTimeout time.Duration, args []string) int {
	if len(args) == 0 {
		out.Errorf("spotify search: missing query")
		return 2
	}
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		out.Errorf("spotify search: missing query")
		return 2
	}

	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg, httpTimeout)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{
		APIBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_API_BASE_URL")),
		HTTP:       spotifyHTTPClient(httpTimeout),
	})
	res, err := api.Search(ctx, accessToken, query, []string{"track", "artist"}, 5)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}
	out.Print(res)
	return 0
}
