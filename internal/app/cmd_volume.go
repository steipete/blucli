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

func cmdVolume(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("volume: missing subcommand (get|set|up|down)")
		return 2
	}

	device, err := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if err != nil {
		out.Errorf("device: %v", err)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch args[0] {
	case "get":
		status, err := client.Status(ctx, bluos.StatusOptions{})
		if err != nil {
			out.Errorf("status: %v", err)
			return 1
		}
		out.Print(map[string]any{"volume": status.Volume, "db": status.DB, "mute": status.Mute})
		return 0
	case "set":
		if len(args) < 2 {
			out.Errorf("volume set: missing level")
			return 2
		}
		level, err := output.ParseIntInRange(args[1], 0, 100)
		if err != nil {
			out.Errorf("volume set: %v", err)
			return 2
		}
		if err := client.VolumeSet(ctx, bluos.VolumeSetOptions{Level: level, TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("volume set: %v", err)
			return 1
		}
		return 0
	case "up":
		if err := client.VolumeDeltaDB(ctx, bluos.VolumeDeltaDBOptions{DeltaDB: 2, TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("volume up: %v", err)
			return 1
		}
		return 0
	case "down":
		if err := client.VolumeDeltaDB(ctx, bluos.VolumeDeltaDBOptions{DeltaDB: -2, TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("volume down: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("volume: unknown subcommand %q", args[0])
		return 2
	}
}
