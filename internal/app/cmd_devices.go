package app

import (
	"context"
	"errors"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/discovery"
	"github.com/steipete/blucli/internal/output"
)

func cmdDevices(ctx context.Context, out *output.Printer, paths config.PathSet, cfg config.Config, cache config.DiscoveryCache, discoverTimeout time.Duration) int {
	_ = cfg

	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()

	devices, err := discovery.Discover(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		out.Errorf("discover: %v", err)
		return 1
	}

	now := time.Now()
	cacheDevices := make([]config.Device, 0, len(devices))
	for _, device := range devices {
		cacheDevices = append(cacheDevices, config.Device{
			ID:   device.ID,
			Host: device.Host,
			Port: device.Port,
			Name: device.Name,
			Type: device.Type,
		})
	}
	newCache := config.NewDiscoveryCache(now, cacheDevices)
	if err := config.SaveDiscoveryCache(paths.CachePath, newCache); err != nil {
		out.Errorf("cache write: %v", err)
		return 1
	}

	if len(devices) == 0 && len(cache.Devices) > 0 {
		out.Warnf("no devices discovered; cache has %d devices (run with longer --discover-timeout)", len(cache.Devices))
	}

	out.Print(devices)
	return 0
}
