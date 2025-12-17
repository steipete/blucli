package discovery

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestDiscoverLSDP_ReceivesAnnounce(t *testing.T) {
	// No t.Parallel: uses UDP port binding.

	tmp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	port := tmp.LocalAddr().(*net.UDPAddr).Port
	_ = tmp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	ctx = WithLSDPPortOverride(ctx, port)

	go func() {
		time.Sleep(25 * time.Millisecond)
		conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
		if err != nil {
			return
		}
		defer conn.Close()

		packet := []byte{6, 76, 83, 68, 80, 1} // [6, 'L','S','D','P', 1]

		nodeID := []byte{0x90, 0x56, 0x82, 0x9F, 0x02, 0x78}
		addr := net.IPv4(127, 0, 0, 1).To4()

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

		_, _ = conn.Write(packet)
	}()

	devs, err := discoverLSDP(ctx)
	if err != nil {
		t.Fatalf("discoverLSDP err = %v", err)
	}
	if len(devs) == 0 {
		t.Fatalf("no devices")
	}
	found := false
	for _, d := range devs {
		if d.ID == "127.0.0.1:11000" && d.Type == "musc" && d.Version == "4.2.1" && d.Source == "lsdp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("devs=%+v", devs)
	}
}
