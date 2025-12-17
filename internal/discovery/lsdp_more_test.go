package discovery

import (
	"testing"
)

func TestBuildLSDPQueryAll(t *testing.T) {
	t.Parallel()

	q := buildLSDPQueryAll()
	if len(q) != 11 {
		t.Fatalf("len=%d; want 11", len(q))
	}
	if string(q[1:5]) != "LSDP" {
		t.Fatalf("hdr=%q", string(q[1:5]))
	}
}

func TestParseLSDP_InvalidPackets(t *testing.T) {
	t.Parallel()

	if _, ok := parseLSDP(nil); ok {
		t.Fatalf("want false")
	}
	if _, ok := parseLSDP([]byte{1, 2, 3}); ok {
		t.Fatalf("want false")
	}
	if _, ok := parseLSDP([]byte{6, 'N', 'O', 'P', 'E', 1}); ok {
		t.Fatalf("want false")
	}
	// Valid header, but no messages.
	if _, ok := parseLSDP([]byte{6, 'L', 'S', 'D', 'P', 1}); ok {
		t.Fatalf("want false")
	}
	// Valid header + broken msg length.
	if _, ok := parseLSDP([]byte{6, 'L', 'S', 'D', 'P', 1, 0}); ok {
		t.Fatalf("want false")
	}
}

func TestParseAnnounce_Invalid(t *testing.T) {
	t.Parallel()

	if _, ok := parseAnnounce([]byte{2, 'A'}); ok {
		t.Fatalf("want false")
	}
	// nodeID len too big
	if _, ok := parseAnnounce([]byte{10, 'A', 5, 1, 2}); ok {
		t.Fatalf("want false")
	}
	// addr len not 4
	msg := []byte{9, 'A', 0, 1, 2, 3, 4, 0, 0}
	if _, ok := parseAnnounce(msg); ok {
		t.Fatalf("want false")
	}
}

func TestParsePort(t *testing.T) {
	t.Parallel()

	if _, err := parsePort("nope"); err == nil {
		t.Fatalf("want error")
	}
	if _, err := parsePort("0"); err == nil {
		t.Fatalf("want error")
	}
	if _, err := parsePort("70000"); err == nil {
		t.Fatalf("want error")
	}
	if p, err := parsePort("11000"); err != nil || p != 11000 {
		t.Fatalf("p=%d err=%v", p, err)
	}
}

func TestClassHelpers(t *testing.T) {
	t.Parallel()

	if !isPlayerClass(0x0001) || !isPlayerClass(0x0008) {
		t.Fatalf("expected player class")
	}
	if isPlayerClass(0x1234) {
		t.Fatalf("unexpected player class")
	}
	if classToType(0x0001) != "musc" || classToType(0x0003) != "musp" || classToType(0x0006) != "musz" || classToType(0x0008) != "mush" {
		t.Fatalf("classToType mismatch")
	}
	if classToType(0x1234) != "" {
		t.Fatalf("want empty")
	}
}

func TestDevicesFromMapSorts(t *testing.T) {
	t.Parallel()

	m := map[string]Device{
		"b": {ID: "b"},
		"a": {ID: "a"},
	}
	devs := devicesFromMap(m)
	if len(devs) != 2 || devs[0].ID != "a" || devs[1].ID != "b" {
		t.Fatalf("devs=%+v", devs)
	}
}
