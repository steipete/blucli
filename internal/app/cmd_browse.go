package app

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdBrowse(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	flags := flag.NewFlagSet("browse", flag.ContinueOnError)
	flags.SetOutput(out.Stderr())

	var key string
	var q string
	var withContext bool
	flags.StringVar(&key, "key", "", "browse key (required)")
	flags.StringVar(&q, "q", "", "search query (optional)")
	flags.BoolVar(&withContext, "context", false, "include context menu items")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(key) == "" {
		out.Errorf("browse: missing --key")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	browse, err := client.Browse(ctx, bluos.BrowseOptions{Key: key, Q: q, WithContextMenuItems: withContext})
	if err != nil {
		out.Errorf("browse: %v", err)
		return 1
	}
	out.Print(browse)
	return 0
}

func cmdPlaylists(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	flags := flag.NewFlagSet("playlists", flag.ContinueOnError)
	flags.SetOutput(out.Stderr())

	var service string
	var category string
	var expr string
	flags.StringVar(&service, "service", "", "service name (optional)")
	flags.StringVar(&category, "category", "", "category (optional)")
	flags.StringVar(&expr, "expr", "", "search expression (optional)")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	playlists, err := client.Playlists(ctx, bluos.PlaylistsOptions{Service: service, Category: category, Expr: expr})
	if err != nil {
		out.Errorf("playlists: %v", err)
		return 1
	}
	out.Print(playlists)
	return 0
}

func cmdInputs(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "", "list":
		rb, err := client.RadioBrowse(ctx, bluos.RadioBrowseOptions{Service: "Capture"})
		if err != nil {
			out.Errorf("inputs: %v", err)
			return 1
		}
		out.Print(rb)
		return 0
	case "play":
		if len(args) < 2 {
			out.Errorf("inputs play: missing id")
			return 2
		}
		rb, err := client.RadioBrowse(ctx, bluos.RadioBrowseOptions{Service: "Capture"})
		if err != nil {
			out.Errorf("inputs: %v", err)
			return 1
		}
		var found *bluos.RadioItem
		for i := range rb.Items {
			if rb.Items[i].ID == args[1] {
				found = &rb.Items[i]
				break
			}
		}
		if found == nil {
			out.Errorf("inputs play: unknown id %q", args[1])
			return 2
		}
		raw, err := url.QueryUnescape(found.URL)
		if err != nil {
			raw = found.URL
		}
		if err := client.Play(ctx, bluos.PlayOptions{URL: raw}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("inputs play: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("inputs: unknown subcommand %q (expected list|play)", sub)
		return 2
	}
}
