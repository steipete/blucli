package app

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/discovery"
	"github.com/steipete/blucli/internal/output"
)

type doctorRow struct {
	ID      string `json:"id"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`

	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	State string `json:"state,omitempty"`
	Name  string `json:"name,omitempty"`
	Model string `json:"model,omitempty"`
}

func cmdDoctor(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, discoverTimeout, httpTimeout time.Duration) int {
	_ = cfg
	_ = cache

	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()

	devices, err := discovery.Discover(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		out.Errorf("doctor discover: %v", err)
		return 1
	}
	if len(devices) == 0 {
		out.Errorf("doctor: no devices discovered")
		return 1
	}

	rows := make([]doctorRow, 0, len(devices))
	for _, d := range devices {
		row := doctorRow{
			ID:      d.ID,
			Host:    d.Host,
			Port:    d.Port,
			Source:  d.Source,
			Version: d.Version,
		}
		dev := config.Device{Host: d.Host, Port: d.Port}
		client := bluos.NewClient(dev.BaseURL(), bluos.Options{Timeout: httpTimeout})
		status, err := client.Status(context.Background(), bluos.StatusOptions{})
		if err != nil {
			row.OK = false
			row.Error = err.Error()
		} else {
			row.OK = true
			row.State = status.State
			row.Name = status.Name
			row.Model = status.Model
		}
		rows = append(rows, row)
	}

	out.Print(rows)
	return 0
}

type diagReport struct {
	Device  config.Device    `json:"device"`
	Status  bluos.Status     `json:"status"`
	Sync    bluos.SyncStatus `json:"sync"`
	Presets bluos.Presets    `json:"presets"`
	Queue   bluos.Playlist   `json:"queue"`
}

func cmdDiag(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer) int {
	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}
	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})

	status, err := client.Status(ctx, bluos.StatusOptions{})
	if err != nil {
		out.Errorf("diag status: %v", err)
		return 1
	}

	sync, err := client.SyncStatus(ctx, bluos.SyncStatusOptions{})
	if err != nil {
		out.Errorf("diag syncstatus: %v", err)
		return 1
	}

	presets, err := client.Presets(ctx)
	if err != nil {
		out.Errorf("diag presets: %v", err)
		return 1
	}

	queue, err := client.Playlist(ctx, bluos.PlaylistOptions{})
	if err != nil {
		out.Errorf("diag queue: %v", err)
		return 1
	}

	report := diagReport{
		Device:  device,
		Status:  status,
		Sync:    sync,
		Presets: presets,
		Queue:   queue,
	}
	out.Print(report)
	return 0
}
