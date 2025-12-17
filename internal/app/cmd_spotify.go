package app

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdSpotify(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("spotify: missing subcommand (login|logout|open|devices|search|play)")
		return 2
	}

	switch args[0] {
	case "login":
		return cmdSpotifyLogin(ctx, out, paths, cfg, args[1:])
	case "logout":
		cfg.Spotify.Token = config.SpotifyToken{}
		if err := config.SaveConfig(paths.ConfigPath, cfg); err != nil {
			out.Errorf("spotify logout: %v", err)
			return 1
		}
		return 0
	case "open":
		device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
		if resolveErr != nil {
			out.Errorf("device: %v", resolveErr)
			return 1
		}
		client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
		if err := client.Play(ctx, bluos.PlayOptions{URL: "Spotify:play"}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("spotify open: %v", err)
			return 1
		}
		return 0
	case "devices":
		return cmdSpotifyDevices(ctx, out, paths, cfg, args[1:])
	case "search":
		return cmdSpotifySearch(ctx, out, paths, cfg, args[1:])
	case "play":
		return cmdSpotifyPlay(ctx, out, paths, cfg, cache, deviceArg, allowDiscover, discoverTimeout, httpTimeout, dryRun, trace, args[1:])
	default:
		out.Errorf("spotify: unknown subcommand %q (expected login|logout|open|devices|search|play)", args[0])
		return 2
	}
}
