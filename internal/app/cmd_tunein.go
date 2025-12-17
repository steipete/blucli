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

func cmdTuneIn(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("tunein: missing subcommand (search|play)")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch args[0] {
	case "search":
		if len(args) < 2 {
			out.Errorf("tunein search: missing query")
			return 2
		}
		query := strings.TrimSpace(strings.Join(args[1:], " "))
		if query == "" {
			out.Errorf("tunein search: missing query")
			return 2
		}
		rb, err := client.RadioBrowse(ctx, bluos.RadioBrowseOptions{Service: "TuneIn", Expr: query})
		if err != nil {
			out.Errorf("tunein search: %v", err)
			return 1
		}
		out.Print(rb)
		return 0
	case "play":
		flags := flag.NewFlagSet("tunein play", flag.ContinueOnError)
		flags.SetOutput(out.Stderr())

		var pick int
		var id string
		flags.IntVar(&pick, "pick", 0, "pick nth audio result (0-based)")
		flags.StringVar(&id, "id", "", "play by TuneIn id (e.g. s21750 or t123)")

		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		rest := flags.Args()

		if strings.TrimSpace(id) != "" {
			u := "TuneIn:" + strings.TrimSpace(id)
			if err := client.Play(ctx, bluos.PlayOptions{URL: u}); err != nil {
				if errors.Is(err, bluos.ErrDryRun) {
					return 0
				}
				out.Errorf("tunein play: %v", err)
				return 1
			}
			return 0
		}

		if len(rest) == 0 {
			out.Errorf("tunein play: missing query (or use --id)")
			return 2
		}
		query := strings.TrimSpace(strings.Join(rest, " "))
		if query == "" {
			out.Errorf("tunein play: missing query (or use --id)")
			return 2
		}

		rb, err := client.RadioBrowse(ctx, bluos.RadioBrowseOptions{Service: "TuneIn", Expr: query})
		if err != nil {
			out.Errorf("tunein play: %v", err)
			return 1
		}

		audio := flattenAudio(rb)
		if len(audio) == 0 {
			out.Errorf("tunein play: no audio results")
			return 1
		}
		if pick < 0 || pick >= len(audio) {
			out.Errorf("tunein play: pick out of range (0..%d): %d", len(audio)-1, pick)
			return 2
		}

		playURL := audio[pick].URL
		if playURL == "" {
			out.Errorf("tunein play: missing URL on result")
			return 1
		}
		decoded, err := url.QueryUnescape(playURL)
		if err != nil {
			decoded = playURL
		}

		if err := client.Play(ctx, bluos.PlayOptions{URL: decoded}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("tunein play: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("tunein: unknown subcommand %q (expected search|play)", args[0])
		return 2
	}
}

func flattenAudio(rb bluos.RadioBrowse) []bluos.RadioItem {
	var out []bluos.RadioItem
	for _, cat := range rb.Categories {
		for _, item := range cat.Items {
			if strings.EqualFold(strings.TrimSpace(item.Type), "audio") {
				out = append(out, item)
			}
		}
	}
	for _, item := range rb.Items {
		if strings.EqualFold(strings.TrimSpace(item.Type), "audio") {
			out = append(out, item)
		}
	}
	return out
}
