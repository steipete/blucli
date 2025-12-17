package discovery

import (
	"context"
	"errors"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
)

type Device struct {
	ID      string `json:"id"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Type    string `json:"type"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source,omitempty"`
}

var serviceTypes = []string{"musc", "musp", "musz", "mush"}

func Discover(ctx context.Context) ([]Device, error) {
	var (
		mdnsDevices []Device
		lsdpDevices []Device
		mdnsErr     error
		lsdpErr     error
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		mdnsDevices, mdnsErr = discoverMDNS(ctx)
	}()
	go func() {
		defer wg.Done()
		lsdpDevices, lsdpErr = discoverLSDP(ctx)
	}()
	wg.Wait()

	// Prefer returning an error only if both mechanisms fail.
	if mdnsErr != nil && lsdpErr != nil {
		return nil, errors.Join(mdnsErr, lsdpErr)
	}

	seen := map[string]Device{}
	for _, d := range mdnsDevices {
		seen[d.ID] = d
	}
	for _, d := range lsdpDevices {
		if existing, ok := seen[d.ID]; ok {
			if existing.Version == "" && d.Version != "" {
				existing.Version = d.Version
			}
			if existing.Type == "" && d.Type != "" {
				existing.Type = d.Type
			}
			if existing.Source == "" {
				existing.Source = d.Source
			} else if d.Source != "" && !strings.Contains(existing.Source, d.Source) {
				existing.Source = existing.Source + "+" + d.Source
			}
			seen[d.ID] = existing
			continue
		}
		seen[d.ID] = d
	}

	devices := make([]Device, 0, len(seen))
	for _, d := range seen {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].ID < devices[j].ID })
	return devices, nil
}

func discoverMDNS(ctx context.Context) ([]Device, error) {
	var resolverErr error
	out := make(chan *zeroconf.ServiceEntry, 64)

	var wg sync.WaitGroup
	for _, t := range serviceTypes {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			// keep going; LSDP may still work
			resolverErr = errors.Join(resolverErr, err)
			continue
		}

		entries := make(chan *zeroconf.ServiceEntry, 32)
		if err := resolver.Browse(ctx, "_"+t+"._tcp", "local.", entries); err != nil {
			resolverErr = errors.Join(resolverErr, err)
			continue
		}

		wg.Add(1)
		go func(ch <-chan *zeroconf.ServiceEntry) {
			defer wg.Done()
			for entry := range ch {
				select {
				case <-ctx.Done():
					return
				case out <- entry:
				}
			}
		}(entries)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	seen := map[string]Device{}

	for entry := range out {
		if entry == nil {
			continue
		}
		device, ok := deviceFromEntry(entry)
		if !ok {
			continue
		}
		device.Source = "mdns"
		if existing, exists := seen[device.ID]; exists {
			if existing.Version == "" && device.Version != "" {
				existing.Version = device.Version
				seen[device.ID] = existing
			}
			continue
		}
		seen[device.ID] = device
	}

	devices := make([]Device, 0, len(seen))
	for _, device := range seen {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].ID < devices[j].ID })
	if len(devices) == 0 && resolverErr != nil {
		return nil, resolverErr
	}
	return devices, nil
}

func deviceFromEntry(entry *zeroconf.ServiceEntry) (Device, bool) {
	if entry.Port == 0 {
		return Device{}, false
	}

	ip := pickIPv4(entry.AddrIPv4)
	if ip == nil {
		return Device{}, false
	}

	host := ip.String()
	id := net.JoinHostPort(host, itoa(entry.Port))

	typ := strings.TrimSuffix(entry.Service, ".")
	typ = strings.TrimPrefix(typ, "_")
	typ = strings.TrimSuffix(typ, "._tcp")

	return Device{
		ID:      id,
		Host:    host,
		Port:    entry.Port,
		Type:    typ,
		Version: parseTXT(entry.Text)["version"],
	}, true
}

func pickIPv4(ips []net.IP) net.IP {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}
	return nil
}

func parseTXT(records []string) map[string]string {
	out := map[string]string{}
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		key, value, ok := strings.Cut(record, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func itoa(i int) string {
	// tiny helper; avoids strconv import on this file.
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [16]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		b[n] = '-'
	}
	return string(b[n:])
}
