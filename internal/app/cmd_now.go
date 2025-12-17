package app

import (
	"context"
	"io"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdNow(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer) int {
	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
	status, err := client.Status(ctx, bluos.StatusOptions{})
	if err != nil {
		out.Errorf("now: %v", err)
		return 1
	}
	out.Print(status)
	return 0
}
