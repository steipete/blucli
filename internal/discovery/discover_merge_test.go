package discovery

import (
	"context"
	"errors"
	"testing"
)

func TestDiscover_JoinsErrorIfBothFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithMDNSOverride(ctx, func(context.Context) ([]Device, error) { return nil, errors.New("mdns down") })
	ctx = WithLSDPOverride(ctx, func(context.Context) ([]Device, error) { return nil, errors.New("lsdp down") })

	_, err := Discover(ctx)
	if err == nil {
		t.Fatalf("want error")
	}
}

func TestDiscover_ReturnsDevicesIfOneMechanismWorks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithMDNSOverride(ctx, func(context.Context) ([]Device, error) {
		return []Device{{ID: "a", Host: "1.1.1.1", Port: 11000, Type: "musc", Version: "1", Source: "mdns"}}, nil
	})
	ctx = WithLSDPOverride(ctx, func(context.Context) ([]Device, error) { return nil, errors.New("lsdp down") })

	devs, err := Discover(ctx)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(devs) != 1 || devs[0].ID != "a" {
		t.Fatalf("devs = %+v", devs)
	}
}

func TestDiscover_MergesDuplicates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithMDNSOverride(ctx, func(context.Context) ([]Device, error) {
		return []Device{
			{ID: "a", Host: "1.1.1.1", Port: 11000, Type: "", Version: "", Source: "mdns"},
			{ID: "b", Host: "2.2.2.2", Port: 11000, Type: "musp", Version: "2", Source: "mdns"},
		}, nil
	})
	ctx = WithLSDPOverride(ctx, func(context.Context) ([]Device, error) {
		return []Device{
			{ID: "a", Host: "1.1.1.1", Port: 11000, Type: "musc", Version: "4", Source: "lsdp"},
		}, nil
	})

	devs, err := Discover(ctx)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("len=%d devs=%+v", len(devs), devs)
	}
	if devs[0].ID != "a" || devs[1].ID != "b" {
		t.Fatalf("ids=%q,%q; want a,b", devs[0].ID, devs[1].ID)
	}
	if devs[0].Version != "4" || devs[0].Type != "musc" {
		t.Fatalf("a=%+v; want version/type merged", devs[0])
	}
	if devs[0].Source != "mdns+lsdp" && devs[0].Source != "lsdp+mdns" {
		t.Fatalf("source=%q; want combined", devs[0].Source)
	}
}
