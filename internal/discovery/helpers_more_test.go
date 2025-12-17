package discovery

import (
	"net"
	"testing"
)

func TestPickIPv4(t *testing.T) {
	t.Parallel()

	if got := pickIPv4(nil); got != nil {
		t.Fatalf("got=%v; want nil", got)
	}
	if got := pickIPv4([]net.IP{net.ParseIP("::1")}); got != nil {
		t.Fatalf("got=%v; want nil", got)
	}
	if got := pickIPv4([]net.IP{nil, net.ParseIP("192.0.2.1")}); got == nil || got.String() != "192.0.2.1" {
		t.Fatalf("got=%v; want 192.0.2.1", got)
	}
}

func TestParseTXT(t *testing.T) {
	t.Parallel()

	m := parseTXT([]string{"version=1", "  nope  ", " =x ", "k= v ", "k2=v2", ""})
	if m["version"] != "1" || m["k"] != "v" || m["k2"] != "v2" {
		t.Fatalf("m=%v", m)
	}
	if _, ok := m["nope"]; ok {
		t.Fatalf("unexpected key")
	}
}

func TestItoa(t *testing.T) {
	t.Parallel()

	if got := itoa(0); got != "0" {
		t.Fatalf("itoa(0)=%q", got)
	}
	if got := itoa(42); got != "42" {
		t.Fatalf("itoa(42)=%q", got)
	}
	if got := itoa(-7); got != "-7" {
		t.Fatalf("itoa(-7)=%q", got)
	}
}
