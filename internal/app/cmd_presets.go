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

func cmdPresets(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("presets: missing subcommand (list|load)")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch args[0] {
	case "list":
		presets, err := client.Presets(ctx)
		if err != nil {
			out.Errorf("presets list: %v", err)
			return 1
		}
		out.Print(presets)
		return 0
	case "load":
		if len(args) < 2 {
			out.Errorf("presets load: missing id")
			return 2
		}
		resp, err := client.LoadPreset(ctx, args[1])
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		if err != nil {
			out.Errorf("presets load: %v", err)
			return 1
		}
		out.Print(resp)
		return 0
	default:
		out.Errorf("presets: unknown subcommand %q", args[0])
		return 2
	}
}
