package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type DiscoveryCache struct {
	UpdatedAt time.Time `json:"updated_at"`
	Devices   []Device  `json:"devices"`
}

func NewDiscoveryCache(updatedAt time.Time, devices []Device) DiscoveryCache {
	out := make([]Device, 0, len(devices))
	for _, device := range devices {
		if device.Host == "" || device.Port == 0 {
			continue
		}
		if device.ID == "" {
			device.ID = fmt.Sprintf("%s:%d", device.Host, device.Port)
		}
		out = append(out, device)
	}
	return DiscoveryCache{UpdatedAt: updatedAt, Devices: out}
}

func LoadDiscoveryCache(path string) (DiscoveryCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DiscoveryCache{}, err
	}

	var cache DiscoveryCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return DiscoveryCache{}, err
	}
	return cache, nil
}

func SaveDiscoveryCache(path string, cache DiscoveryCache) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (c DiscoveryCache) Lookup(idOrHostPort string) (Device, bool) {
	for _, device := range c.Devices {
		if device.ID == idOrHostPort {
			return device, true
		}
		if fmt.Sprintf("%s:%d", device.Host, device.Port) == idOrHostPort {
			return device, true
		}
	}
	return Device{}, false
}
