package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"unicode"

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

		if likelyNameArg(raw) {
			if matches := cache.FindByName(raw); len(matches) == 1 {
				return matches[0], nil
			} else if len(matches) > 1 {
				return config.Device{}, fmt.Errorf("ambiguous device name %q; matches: %s", raw, formatCandidates(matches))
			}

			if allowDiscover {
				ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
				defer cancel()

				devices, derr := discovery.Discover(ctx)
				if derr != nil && !errors.Is(derr, context.DeadlineExceeded) {
					return config.Device{}, derr
				}
				if d, ok := matchByName(raw, devices); ok {
					return config.Device{ID: d.ID, Host: d.Host, Port: d.Port, Name: d.Name, Type: d.Type}, nil
				}
				if ms := matchManyByName(raw, devices); len(ms) > 1 {
					return config.Device{}, fmt.Errorf("ambiguous device name %q; matches: %s", raw, formatDiscoveryCandidates(ms))
				}
			}
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

func matchByName(query string, devices []discovery.Device) (discovery.Device, bool) {
	ms := matchManyByName(query, devices)
	if len(ms) != 1 {
		return discovery.Device{}, false
	}
	return ms[0], true
}

func matchManyByName(query string, devices []discovery.Device) []discovery.Device {
	q := normalizeDeviceName(query)
	if q == "" {
		return nil
	}

	var exact []discovery.Device
	var fuzzy []discovery.Device
	for _, d := range devices {
		n := normalizeDeviceName(d.Name)
		if n == "" {
			continue
		}
		if n == q {
			exact = append(exact, d)
			continue
		}
		if strings.Contains(n, q) || strings.Contains(q, n) {
			fuzzy = append(fuzzy, d)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return fuzzy
}

func normalizeDeviceName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func likelyNameArg(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	if strings.Contains(s, ":") || strings.Contains(s, "/") {
		return false
	}
	if strings.ContainsAny(s, " \t") {
		return true
	}
	if strings.Contains(s, ".") {
		return false
	}
	if net.ParseIP(strings.Trim(s, "[]")) != nil {
		return false
	}
	return true
}

func formatCandidates(devices []config.Device) string {
	var parts []string
	for _, d := range devices {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			name = d.ID
		}
		parts = append(parts, fmt.Sprintf("%s (%s:%d)", name, d.Host, d.Port))
	}
	return strings.Join(parts, ", ")
}

func formatDiscoveryCandidates(devices []discovery.Device) string {
	var parts []string
	for _, d := range devices {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			name = d.ID
		}
		parts = append(parts, fmt.Sprintf("%s (%s:%d)", name, d.Host, d.Port))
	}
	return strings.Join(parts, ", ")
}
