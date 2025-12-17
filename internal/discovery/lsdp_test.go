package discovery

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseLSDPAnnounce(t *testing.T) {
	t.Parallel()

	// Packet header: [6, 'L','S','D','P', 1]
	packet := []byte{6, 76, 83, 68, 80, 1}

	// Announce message:
	// len, 'A', nodeIDLen, nodeID, addrLen, addr(4), count,
	// class(2), txtCount, keyLen, key, valLen, val, ...
	nodeID := []byte{0x90, 0x56, 0x82, 0x9F, 0x02, 0x78}
	addr := net.IPv4(192, 168, 1, 10).To4()

	msg := make([]byte, 0, 64)
	msg = append(msg, 0)   // placeholder len
	msg = append(msg, 'A') // type
	msg = append(msg, byte(len(nodeID)))
	msg = append(msg, nodeID...)
	msg = append(msg, 4)
	msg = append(msg, addr...)
	msg = append(msg, 1) // one record

	class := make([]byte, 2)
	binary.BigEndian.PutUint16(class, 0x0001)
	msg = append(msg, class...)
	msg = append(msg, 2) // two txt records

	msg = append(msg, byte(len("port")))
	msg = append(msg, []byte("port")...)
	msg = append(msg, byte(len("11000")))
	msg = append(msg, []byte("11000")...)

	msg = append(msg, byte(len("version")))
	msg = append(msg, []byte("version")...)
	msg = append(msg, byte(len("4.2.1")))
	msg = append(msg, []byte("4.2.1")...)

	msg[0] = byte(len(msg))
	packet = append(packet, msg...)

	announces, ok := parseLSDP(packet)
	if !ok {
		t.Fatalf("parseLSDP ok=false")
	}
	if len(announces) != 1 {
		t.Fatalf("announces=%d", len(announces))
	}
	if announces[0].Address.String() != "192.168.1.10" {
		t.Fatalf("addr=%q", announces[0].Address.String())
	}
	if len(announces[0].Records) != 1 {
		t.Fatalf("records=%d", len(announces[0].Records))
	}
	rec := announces[0].Records[0]
	if rec.Class != 0x0001 {
		t.Fatalf("class=0x%04x", rec.Class)
	}
	if rec.TXT["port"] != "11000" || rec.TXT["version"] != "4.2.1" {
		t.Fatalf("txt=%v", rec.TXT)
	}
}
