package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdWatch(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("watch: missing type (status|sync)")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	clientTimeout := httpTimeout
	if clientTimeout < 40*time.Second {
		clientTimeout = 40 * time.Second
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: clientTimeout, DryRun: dryRun, Trace: trace})

	switch args[0] {
	case "status":
		return watchStatus(ctx, out, client)
	case "sync":
		return watchSync(ctx, out, client)
	default:
		out.Errorf("watch: unknown type %q (expected status|sync)", args[0])
		return 2
	}
}

func watchStatus(ctx context.Context, out *output.Printer, client *bluos.Client) int {
	var lastETag string
	for {
		select {
		case <-ctx.Done():
			return 0
		default:
		}

		status, err := client.Status(ctx, bluos.StatusOptions{TimeoutSeconds: 30, ETag: lastETag})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		if err != nil {
			out.Errorf("watch status: %v", err)
			return 1
		}
		if status.ETag == "" || status.ETag != lastETag {
			lastETag = status.ETag
			out.Print(status)
		}
	}
}

func watchSync(ctx context.Context, out *output.Printer, client *bluos.Client) int {
	var lastETag string
	for {
		select {
		case <-ctx.Done():
			return 0
		default:
		}

		sync, err := client.SyncStatus(ctx, bluos.SyncStatusOptions{TimeoutSeconds: 30, ETag: lastETag})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		if err != nil {
			out.Errorf("watch sync: %v", err)
			return 1
		}
		if sync.ETag == "" || sync.ETag != lastETag {
			lastETag = sync.ETag
			out.Print(sync)
		}
	}
}

func cmdSleep(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer) int {
	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
	minutes, err := client.Sleep(ctx)
	if errors.Is(err, bluos.ErrDryRun) {
		return 0
	}
	if err != nil {
		out.Errorf("sleep: %v", err)
		return 1
	}
	fmt.Fprintln(out.Stdout(), minutes)
	return 0
}
