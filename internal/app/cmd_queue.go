package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdQueue(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("queue: missing subcommand (list|clear|delete|move|save)")
		return 2
	}

	var start, end int
	var hasStart, hasEnd bool

	sub := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--start" || args[i] == "-start":
			if i+1 >= len(args) {
				out.Errorf("queue: --start requires value")
				return 2
			}
			v, err := output.ParseIntInRange(args[i+1], 0, 1<<30)
			if err != nil {
				out.Errorf("queue: --start: %v", err)
				return 2
			}
			start, hasStart = v, true
			i++
		case strings.HasPrefix(args[i], "--start="):
			v, err := output.ParseIntInRange(strings.TrimPrefix(args[i], "--start="), 0, 1<<30)
			if err != nil {
				out.Errorf("queue: --start: %v", err)
				return 2
			}
			start, hasStart = v, true
		case args[i] == "--end" || args[i] == "-end":
			if i+1 >= len(args) {
				out.Errorf("queue: --end requires value")
				return 2
			}
			v, err := output.ParseIntInRange(args[i+1], 0, 1<<30)
			if err != nil {
				out.Errorf("queue: --end: %v", err)
				return 2
			}
			end, hasEnd = v, true
			i++
		case strings.HasPrefix(args[i], "--end="):
			v, err := output.ParseIntInRange(strings.TrimPrefix(args[i], "--end="), 0, 1<<30)
			if err != nil {
				out.Errorf("queue: --end: %v", err)
				return 2
			}
			end, hasEnd = v, true
		default:
			sub = append(sub, args[i])
		}
	}
	if len(sub) == 0 {
		out.Errorf("queue: missing subcommand (list|clear|delete|move|save)")
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch sub[0] {
	case "list":
		opts := bluos.PlaylistOptions{}
		if hasStart {
			opts.Start = &start
		}
		if hasEnd {
			opts.End = &end
		}
		pl, err := client.Playlist(ctx, opts)
		if err != nil {
			out.Errorf("queue list: %v", err)
			return 1
		}
		out.Print(pl)
		return 0
	case "clear":
		pl, err := client.Clear(ctx)
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		if err != nil {
			out.Errorf("queue clear: %v", err)
			return 1
		}
		out.Print(pl)
		return 0
	case "delete":
		if len(sub) < 2 {
			out.Errorf("queue delete: missing id")
			return 2
		}
		id, err := output.ParseIntInRange(sub[1], 0, 1<<30)
		if err != nil {
			out.Errorf("queue delete: %v", err)
			return 2
		}
		pl, err := client.Delete(ctx, id)
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		if err != nil {
			out.Errorf("queue delete: %v", err)
			return 1
		}
		out.Print(pl)
		return 0
	case "move":
		if len(sub) < 3 {
			out.Errorf("queue move: usage: queue move <old> <new>")
			return 2
		}
		oldID, err := output.ParseIntInRange(sub[1], 0, 1<<30)
		if err != nil {
			out.Errorf("queue move: %v", err)
			return 2
		}
		newID, err := output.ParseIntInRange(sub[2], 0, 1<<30)
		if err != nil {
			out.Errorf("queue move: %v", err)
			return 2
		}
		pl, err := client.Move(ctx, oldID, newID)
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		if err != nil {
			out.Errorf("queue move: %v", err)
			return 1
		}
		out.Print(pl)
		return 0
	case "save":
		if len(sub) < 2 {
			out.Errorf("queue save: missing name")
			return 2
		}
		name := strings.Join(sub[1:], " ")
		name = strings.TrimSpace(name)
		if name == "" {
			out.Errorf("queue save: missing name")
			return 2
		}
		resp, err := client.Save(ctx, name)
		if errors.Is(err, bluos.ErrDryRun) {
			return 0
		}
		if err != nil {
			out.Errorf("queue save: %v", err)
			return 1
		}
		out.Print(resp)
		return 0
	default:
		out.Errorf("queue: unknown subcommand %q", sub[0])
		return 2
	}
}
