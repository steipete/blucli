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

func cmdMute(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("mute: missing subcommand (on|off|toggle)")
		return 2
	}

	device, err := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if err != nil {
		out.Errorf("device: %v", err)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch args[0] {
	case "on":
		if err := client.VolumeMute(ctx, bluos.VolumeMuteOptions{Mute: true, TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("mute on: %v", err)
			return 1
		}
		return 0
	case "off":
		if err := client.VolumeMute(ctx, bluos.VolumeMuteOptions{Mute: false, TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("mute off: %v", err)
			return 1
		}
		return 0
	case "toggle":
		status, err := client.Status(ctx, bluos.StatusOptions{})
		if err != nil {
			out.Errorf("status: %v", err)
			return 1
		}
		if err := client.VolumeMute(ctx, bluos.VolumeMuteOptions{Mute: !bool(status.Mute), TellSlaves: true}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("mute toggle: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("mute: unknown subcommand %q", args[0])
		return 2
	}
}
