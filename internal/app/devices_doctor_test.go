package app

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/discovery"
	"github.com/steipete/blucli/internal/output"
)

func TestCmdDevices_WritesCacheAndPrints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = discovery.WithMDNSOverride(ctx, func(context.Context) ([]discovery.Device, error) {
		return []discovery.Device{{ID: "d1", Host: "127.0.0.1", Port: 11000, Type: "musc", Version: "1.2.3"}}, nil
	})
	ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })

	paths := config.PathSet{CachePath: filepath.Join(t.TempDir(), "cache.json")}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})

	code := cmdDevices(ctx, out, paths, config.Config{}, config.DiscoveryCache{}, 250*time.Millisecond)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "d1 (musc)") || !strings.Contains(got, "v1.2.3") {
		t.Fatalf("stdout = %q", got)
	}

	loaded, err := config.LoadDiscoveryCache(paths.CachePath)
	if err != nil {
		t.Fatalf("LoadDiscoveryCache: %v", err)
	}
	if len(loaded.Devices) != 1 || loaded.Devices[0].Host != "127.0.0.1" {
		t.Fatalf("cache=%+v", loaded)
	}
}

func TestCmdDevices_WarnsWhenDiscoveryEmptyButCacheHasDevices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = discovery.WithMDNSOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })
	ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })

	paths := config.PathSet{CachePath: filepath.Join(t.TempDir(), "cache.json")}
	cache := config.NewDiscoveryCache(time.Now(), []config.Device{{Host: "127.0.0.1", Port: 11000}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})

	code := cmdDevices(ctx, out, paths, config.Config{}, cache, 250*time.Millisecond)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "warn: no devices discovered") {
		t.Fatalf("stderr = %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "no devices") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestCmdDoctor_StubbedDiscoveryAndStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<status state="play" volume="10" mute="0" name="Room" model="M"/>`))
	}))
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	ctx := context.Background()
	ctx = discovery.WithMDNSOverride(ctx, func(context.Context) ([]discovery.Device, error) {
		return []discovery.Device{{ID: host + ":" + portStr, Host: host, Port: port, Source: "mdns"}}, nil
	})
	ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})

	code := cmdDoctor(ctx, out, config.Config{}, config.DiscoveryCache{}, 250*time.Millisecond, 2*time.Second)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "\"ok\": true") || !strings.Contains(got, "\"name\": \"Room\"") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestCmdDoctor_NoDevices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = discovery.WithMDNSOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })
	ctx = discovery.WithLSDPOverride(ctx, func(context.Context) ([]discovery.Device, error) { return nil, nil })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := output.New(output.Options{Stdout: &stdout, Stderr: &stderr})

	code := cmdDoctor(ctx, out, config.Config{}, config.DiscoveryCache{}, 250*time.Millisecond, 250*time.Millisecond)
	if code != 1 {
		t.Fatalf("code=%d; want 1", code)
	}
	if got := stderr.String(); !strings.Contains(got, "doctor: no devices discovered") {
		t.Fatalf("stderr = %q", got)
	}
}
