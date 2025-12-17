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

func cmdPlayback(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, verb string) int {
	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	var err error
	switch verb {
	case "play":
		err = client.Play(ctx, bluos.PlayOptions{})
	case "pause":
		err = client.Pause(ctx, bluos.PauseOptions{})
	case "stop":
		err = client.Stop(ctx)
	case "next":
		err = client.Skip(ctx)
	case "prev":
		err = client.Back(ctx)
	default:
		out.Errorf("unknown playback command: %q", verb)
		return 2
	}

	if errors.Is(err, bluos.ErrDryRun) {
		return 0
	}
	if err != nil {
		out.Errorf("%s: %v", verb, err)
		return 1
	}
	return 0
}

func cmdShuffle(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("shuffle: missing arg (on|off)")
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
		if err := client.Shuffle(ctx, true); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("shuffle on: %v", err)
			return 1
		}
		return 0
	case "off":
		if err := client.Shuffle(ctx, false); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("shuffle off: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("shuffle: unknown arg %q (expected on|off)", args[0])
		return 2
	}
}

func cmdRepeat(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("repeat: missing arg (off|track|queue)")
		return 2
	}

	device, err := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if err != nil {
		out.Errorf("device: %v", err)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	state, ok := map[string]int{
		"queue": 0,
		"track": 1,
		"off":   2,
	}[args[0]]
	if !ok {
		out.Errorf("repeat: unknown arg %q (expected off|track|queue)", args[0])
		return 2
	}

	if err := client.Repeat(ctx, state); err != nil {
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		out.Errorf("repeat: %v", err)
		return 1
	}
	return 0
}
