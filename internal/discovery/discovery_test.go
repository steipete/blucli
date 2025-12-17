package discovery

import (
	"net"
	"testing"

	"github.com/grandcat/zeroconf"
)

func TestDeviceFromEntry(t *testing.T) {
	t.Parallel()

	entry := &zeroconf.ServiceEntry{
		ServiceRecord: zeroconf.ServiceRecord{
			Service: "_musc._tcp",
		},
		Port:     11000,
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.10")},
		Text:     []string{"version=4.2.1"},
	}

	device, ok := deviceFromEntry(entry)
	if !ok {
		t.Fatalf("deviceFromEntry() ok = false")
	}
	if device.Type != "musc" {
		t.Fatalf("Type = %q; want musc", device.Type)
	}
	if device.Version != "4.2.1" {
		t.Fatalf("Version = %q; want 4.2.1", device.Version)
	}
	if device.ID != "192.168.1.10:11000" {
		t.Fatalf("ID = %q; want 192.168.1.10:11000", device.ID)
	}
}
