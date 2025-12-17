package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDiscoveryCache_FiltersAndDefaultsID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cache := NewDiscoveryCache(now, []Device{
		{Host: "", Port: 11000},             // drop
		{Host: "127.0.0.1", Port: 0},        // drop
		{Host: "127.0.0.1", Port: 11000},    // keep, id default
		{ID: "x", Host: "1.2.3.4", Port: 1}, // keep
	})
	if len(cache.Devices) != 2 {
		t.Fatalf("devices=%+v", cache.Devices)
	}
	if cache.Devices[0].ID == "" || cache.Devices[0].Host == "" {
		t.Fatalf("device=%+v", cache.Devices[0])
	}
}

func TestDiscoveryCache_SaveLoadLookup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.json")
	cache := NewDiscoveryCache(time.Now(), []Device{{Host: "127.0.0.1", Port: 11000}})
	if err := SaveDiscoveryCache(path, cache); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat: %v", err)
	}

	loaded, err := LoadDiscoveryCache(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Devices) != 1 {
		t.Fatalf("loaded=%+v", loaded)
	}

	if _, ok := loaded.Lookup("nope"); ok {
		t.Fatalf("unexpected hit")
	}
	if d, ok := loaded.Lookup(loaded.Devices[0].ID); !ok || d.Host != "127.0.0.1" {
		t.Fatalf("lookup by id: ok=%v d=%+v", ok, d)
	}
	if d, ok := loaded.Lookup("127.0.0.1:11000"); !ok || d.Host != "127.0.0.1" {
		t.Fatalf("lookup by host:port: ok=%v d=%+v", ok, d)
	}
}

func TestDiscoveryCache_FindByName(t *testing.T) {
	t.Parallel()

	cache := NewDiscoveryCache(time.Now(), []Device{
		{Host: "127.0.0.1", Port: 11000, Name: "Schlafzimmer"},
		{Host: "127.0.0.2", Port: 11000, Name: "Kitchen"},
		{Host: "127.0.0.3", Port: 11000, Name: "Kitchen"}, // duplicate name
	})

	if ms := cache.FindByName("schlafzimmer"); len(ms) != 1 || ms[0].Host != "127.0.0.1" {
		t.Fatalf("matches=%v", ms)
	}

	if ms := cache.FindByName("kit"); len(ms) != 2 {
		t.Fatalf("matches=%v", ms)
	}

	if ms := cache.FindByName("  "); len(ms) != 0 {
		t.Fatalf("matches=%v", ms)
	}
}

func TestLoadDiscoveryCache_InvalidJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadDiscoveryCache(path); err == nil {
		t.Fatalf("want error")
	}
}

func TestSaveDiscoveryCache_ParentDirError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := filepath.Join(blocker, "cache.json")
	if err := SaveDiscoveryCache(path, DiscoveryCache{}); err == nil {
		t.Fatalf("want error")
	}
}
