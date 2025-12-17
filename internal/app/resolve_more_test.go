package app

import (
	"context"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/discovery"
)

func TestResolveDevice_ArgAliasAndCache(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DefaultDevice: "",
		Aliases:       map[string]string{"a": "192.0.2.1:12000"},
	}
	cache := config.NewDiscoveryCache(time.Now(), []config.Device{{ID: "c1", Host: "127.0.0.1", Port: 11000}})

	d, err := resolveDevice(context.Background(), cfg, cache, "a", false, 0)
	if err != nil || d.Host != "192.0.2.1" || d.Port != 12000 {
		t.Fatalf("d=%+v err=%v", d, err)
	}

	d, err = resolveDevice(context.Background(), cfg, cache, "c1", false, 0)
	if err != nil || d.Host != "127.0.0.1" || d.Port != 11000 {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestResolveDevice_UsesEnvBLUDevice(t *testing.T) {
	t.Setenv("BLU_DEVICE", "192.0.2.2:11001")

	cfg := config.Config{Aliases: map[string]string{}}
	d, err := resolveDevice(context.Background(), cfg, config.DiscoveryCache{}, "", false, 0)
	if err != nil || d.Host != "192.0.2.2" || d.Port != 11001 {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestResolveDevice_NoDeviceSelectedWithoutDiscover(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Aliases: map[string]string{}}
	_, err := resolveDevice(context.Background(), cfg, config.DiscoveryCache{}, "", false, 0)
	if err == nil {
		t.Fatalf("want error")
	}
}

func TestResolveDevice_UsesSingleCachedDevice(t *testing.T) {
	t.Parallel()

	cache := config.NewDiscoveryCache(time.Now(), []config.Device{{Host: "127.0.0.1", Port: 11000}})
	cfg := config.Config{Aliases: map[string]string{}}
	d, err := resolveDevice(context.Background(), cfg, cache, "", false, 0)
	if err != nil || d.Host != "127.0.0.1" {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestResolveDevice_DiscoveryPaths(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Aliases: map[string]string{}}

	{
		ctx := discovery.WithMDNSOverride(context.Background(), func(context.Context) ([]discovery.Device, error) {
			return []discovery.Device{{ID: "a", Host: "127.0.0.1", Port: 11000}}, nil
		})
		ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })
		d, err := resolveDevice(ctx, cfg, config.DiscoveryCache{}, "", true, 250*time.Millisecond)
		if err != nil || d.Host != "127.0.0.1" {
			t.Fatalf("d=%+v err=%v", d, err)
		}
	}

	{
		ctx := discovery.WithMDNSOverride(context.Background(), func(context.Context) ([]discovery.Device, error) { return nil, nil })
		ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })
		if _, err := resolveDevice(ctx, cfg, config.DiscoveryCache{}, "", true, 250*time.Millisecond); err == nil {
			t.Fatalf("want error")
		}
	}

	{
		ctx := discovery.WithMDNSOverride(context.Background(), func(context.Context) ([]discovery.Device, error) {
			return []discovery.Device{
				{ID: "a", Host: "127.0.0.1", Port: 11000},
				{ID: "b", Host: "127.0.0.2", Port: 11000},
			}, nil
		})
		ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })
		if _, err := resolveDevice(ctx, cfg, config.DiscoveryCache{}, "", true, 250*time.Millisecond); err == nil {
			t.Fatalf("want error")
		}
	}
}
