package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/discovery"
)

func resolveDevice(ctx context.Context, cfg config.Config, cache config.DiscoveryCache, arg string, allowDiscover bool, discoverTimeout time.Duration) (config.Device, error) {
	raw := strings.TrimSpace(arg)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("BLU_DEVICE"))
	}
	if raw == "" {
		raw = strings.TrimSpace(cfg.DefaultDevice)
	}

	if raw != "" {
		if resolved, ok := cfg.Aliases[raw]; ok {
			raw = resolved
		}
		if fromCache, ok := cache.Lookup(raw); ok {
			return fromCache, nil
		}
		device, err := config.ParseDevice(raw)
		if err == nil {
			return device, nil
		}
		return config.Device{}, fmt.Errorf("unable to resolve %q (set --device or BLU_DEVICE)", raw)
	}

	if len(cache.Devices) == 1 {
		return cache.Devices[0], nil
	}

	if !allowDiscover {
		return config.Device{}, errors.New("no device selected")
	}

	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()

	devices, err := discovery.Discover(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return config.Device{}, err
	}
	if len(devices) == 1 {
		return config.Device{ID: devices[0].ID, Host: devices[0].Host, Port: devices[0].Port}, nil
	}
	if len(devices) == 0 {
		return config.Device{}, errors.New("no devices discovered (run `blu devices` or set --device)")
	}
	return config.Device{}, fmt.Errorf("multiple devices discovered (%d); pick one with --device", len(devices))
}
