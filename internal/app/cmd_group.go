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

func cmdGroup(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	if len(args) == 0 {
		out.Errorf("group: missing subcommand (status|add|remove)")
		return 2
	}

	var groupName string
	subArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--name":
			if i+1 >= len(args) {
				out.Errorf("group: --name requires value")
				return 2
			}
			groupName = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--name="):
			groupName = strings.TrimPrefix(args[i], "--name=")
		default:
			subArgs = append(subArgs, args[i])
		}
	}

	if len(subArgs) == 0 {
		out.Errorf("group: missing subcommand (status|add|remove)")
		return 2
	}

	device, err := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if err != nil {
		out.Errorf("device: %v", err)
		return 1
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	switch subArgs[0] {
	case "status":
		sync, err := client.SyncStatus(ctx, bluos.SyncStatusOptions{})
		if err != nil {
			out.Errorf("syncstatus: %v", err)
			return 1
		}
		out.Print(sync)
		return 0
	case "add":
		if len(subArgs) < 2 {
			out.Errorf("group add: missing slave (host[:port] or alias)")
			return 2
		}
		slave, err := resolveDevice(ctx, cfg, cache, subArgs[1], allowDiscover, discoverTimeout)
		if err != nil {
			out.Errorf("group add: %v", err)
			return 1
		}
		if err := client.AddSlave(ctx, bluos.AddSlaveOptions{SlaveHost: slave.Host, SlavePort: slave.Port, GroupName: groupName}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("group add: %v", err)
			return 1
		}
		return 0
	case "remove":
		if len(subArgs) < 2 {
			out.Errorf("group remove: missing slave (host[:port] or alias)")
			return 2
		}
		slave, err := resolveDevice(ctx, cfg, cache, subArgs[1], allowDiscover, discoverTimeout)
		if err != nil {
			out.Errorf("group remove: %v", err)
			return 1
		}
		if err := client.RemoveSlave(ctx, bluos.RemoveSlaveOptions{SlaveHost: slave.Host, SlavePort: slave.Port}); err != nil {
			if errors.Is(err, bluos.ErrDryRun) {
				return 0
			}
			out.Errorf("group remove: %v", err)
			return 1
		}
		return 0
	default:
		out.Errorf("group: unknown subcommand %q", subArgs[0])
		return 2
	}
}
