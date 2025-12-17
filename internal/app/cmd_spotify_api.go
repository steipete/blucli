package app

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
	"github.com/steipete/blucli/internal/spotify"
)

func cmdSpotifyDevices(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, args []string) int {
	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{
		APIBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_API_BASE_URL")),
		HTTP:       http.DefaultClient,
	})
	devs, err := api.Devices(ctx, accessToken)
	if err != nil {
		out.Errorf("spotify devices: %v", err)
		return 1
	}
	out.Print(devs)
	return 0
}

func cmdSpotifySearch(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, args []string) int {
	if len(args) == 0 {
		out.Errorf("spotify search: missing query")
		return 2
	}
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		out.Errorf("spotify search: missing query")
		return 2
	}

	accessToken, _, err := spotifyAccessToken(ctx, paths, cfg)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}

	api := spotify.NewAPI(spotify.APIOptions{
		APIBaseURL: strings.TrimSpace(os.Getenv("BLU_SPOTIFY_API_BASE_URL")),
		HTTP:       http.DefaultClient,
	})
	res, err := api.Search(ctx, accessToken, query, []string{"track", "artist"}, 5)
	if err != nil {
		out.Errorf("spotify search: %v", err)
		return 1
	}
	out.Print(res)
	return 0
}
