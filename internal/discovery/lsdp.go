package discovery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
	"time"
)

const lsdpPort = 11430

var lsdpClasses = []uint16{0x0001, 0x0003, 0x0006, 0x0008}

type lsdpPortOverrideKey struct{}

func WithLSDPPortOverride(ctx context.Context, port int) context.Context {
	return context.WithValue(ctx, lsdpPortOverrideKey{}, port)
}

func discoverLSDP(ctx context.Context) ([]Device, error) {
	port := lsdpPort
	if v := ctx.Value(lsdpPortOverrideKey{}); v != nil {
		if p, ok := v.(int); ok && p > 0 && p <= 65535 {
			port = p
		}
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		// Port may be in use (BluOS Controller) or blocked; treat as soft failure.
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetReadBuffer(1 << 20)
	_ = conn.SetWriteBuffer(1 << 20)

	broadcastIPs := interfaceBroadcastIPs()
	query := buildLSDPQueryAll()
	// Startup timing per doc: 7 packets at [0,1,2,3,5,7,10]s + random.
	go sendLSDPQueries(ctx, conn, port, query, broadcastIPs, lsdpSendHooks{})

	seen := map[string]Device{}
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return devicesFromMap(seen), nil
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			// stop on permanent read error
			return devicesFromMap(seen), nil
		}

		packet := buf[:n]
		msgs, ok := parseLSDP(packet)
		if !ok {
			continue
		}
		for _, announce := range msgs {
			if announce.Address == nil {
				continue
			}
			for _, record := range announce.Records {
				if !isPlayerClass(record.Class) {
					continue
				}
				port := 11000
				if p, ok := record.TXT["port"]; ok {
					if parsed, perr := parsePort(p); perr == nil {
						port = parsed
					}
				}
				name := lsdpDeviceName(record.TXT)
				host := announce.Address.String()
				id := net.JoinHostPort(host, fmt.Sprintf("%d", port))
				if _, exists := seen[id]; exists {
					continue
				}
				seen[id] = Device{
					ID:      id,
					Host:    host,
					Port:    port,
					Name:    name,
					Type:    classToType(record.Class),
					Version: record.TXT["version"],
					Source:  "lsdp",
				}
			}
		}
	}
}

type lsdpAnnounce struct {
	NodeID  []byte
	Address net.IP
	Records []lsdpRecord
}

type lsdpRecord struct {
	Class uint16
	TXT   map[string]string
}

func parseLSDP(packet []byte) ([]lsdpAnnounce, bool) {
	if len(packet) < 6 {
		return nil, false
	}
	hdrLen := int(packet[0])
	if hdrLen < 6 || hdrLen > len(packet) {
		return nil, false
	}
	if string(packet[1:5]) != "LSDP" {
		return nil, false
	}
	// version := packet[5]

	p := hdrLen
	var announces []lsdpAnnounce
	for p < len(packet) {
		if p+1 > len(packet) {
			break
		}
		msgLen := int(packet[p])
		if msgLen <= 0 || p+msgLen > len(packet) {
			break
		}
		msg := packet[p : p+msgLen]
		p += msgLen

		if len(msg) < 2 {
			continue
		}
		switch msg[1] {
		case 'A':
			ann, ok := parseAnnounce(msg)
			if ok {
				announces = append(announces, ann)
			}
		default:
		}
	}
	if len(announces) == 0 {
		return nil, false
	}
	return announces, true
}

func parseAnnounce(msg []byte) (lsdpAnnounce, bool) {
	// msg[0]=len, msg[1]='A'
	i := 2
	if i >= len(msg) {
		return lsdpAnnounce{}, false
	}
	nodeIDLen := int(msg[i])
	i++
	if i+nodeIDLen > len(msg) {
		return lsdpAnnounce{}, false
	}
	nodeID := append([]byte(nil), msg[i:i+nodeIDLen]...)
	i += nodeIDLen

	if i >= len(msg) {
		return lsdpAnnounce{}, false
	}
	addrLen := int(msg[i])
	i++
	if addrLen != 4 || i+addrLen > len(msg) {
		return lsdpAnnounce{}, false
	}
	address := net.IPv4(msg[i], msg[i+1], msg[i+2], msg[i+3])
	i += addrLen

	if i >= len(msg) {
		return lsdpAnnounce{}, false
	}
	count := int(msg[i])
	i++

	records := make([]lsdpRecord, 0, count)
	for r := 0; r < count; r++ {
		if i+2 > len(msg) {
			return lsdpAnnounce{}, false
		}
		class := binary.BigEndian.Uint16(msg[i : i+2])
		i += 2

		if i >= len(msg) {
			return lsdpAnnounce{}, false
		}
		txtCount := int(msg[i])
		i++
		txt := map[string]string{}
		for t := 0; t < txtCount; t++ {
			if i >= len(msg) {
				return lsdpAnnounce{}, false
			}
			keyLen := int(msg[i])
			i++
			if i+keyLen > len(msg) {
				return lsdpAnnounce{}, false
			}
			key := string(msg[i : i+keyLen])
			i += keyLen

			if i >= len(msg) {
				return lsdpAnnounce{}, false
			}
			valLen := int(msg[i])
			i++
			if i+valLen > len(msg) {
				return lsdpAnnounce{}, false
			}
			val := string(msg[i : i+valLen])
			i += valLen
			if key != "" {
				txt[key] = val
			}
		}
		records = append(records, lsdpRecord{Class: class, TXT: txt})
	}

	return lsdpAnnounce{NodeID: nodeID, Address: address, Records: records}, true
}

func buildLSDPQueryAll() []byte {
	// Mirrors BluOS Controller app:
	// [6, 'L','S','D','P', 1, 5, 'Q', 1, 0xFF, 0xFF]
	return []byte{6, 76, 83, 68, 80, 1, 5, 'Q', 1, 0xFF, 0xFF}
}

func isPlayerClass(class uint16) bool {
	for _, c := range lsdpClasses {
		if class == c {
			return true
		}
	}
	return false
}

func classToType(class uint16) string {
	switch class {
	case 0x0001:
		return "musc"
	case 0x0003:
		return "musp"
	case 0x0006:
		return "musz"
	case 0x0008:
		return "mush"
	default:
		return ""
	}
}

func lsdpDeviceName(txt map[string]string) string {
	for _, k := range []string{"name", "device", "devname", "playername", "player", "friendly_name", "label"} {
		if v := txt[k]; v != "" {
			return v
		}
	}
	return ""
}

func interfaceBroadcastIPs() []net.IP {
	var out []net.IP
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, ipnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || ipnet == nil || ipnet.Mask == nil || len(ipnet.Mask) != 4 {
				continue
			}
			bcast := make(net.IP, 4)
			for i := 0; i < 4; i++ {
				bcast[i] = ip4[i] | ^ipnet.Mask[i]
			}
			out = append(out, bcast)
		}
	}
	if len(out) == 0 {
		out = append(out, net.IPv4bcast)
	}
	return out
}

func devicesFromMap(m map[string]Device) []Device {
	devices := make([]Device, 0, len(m))
	for _, d := range m {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].ID < devices[j].ID })
	return devices
}

func parsePort(s string) (int, error) {
	var p int
	_, err := fmt.Sscanf(s, "%d", &p)
	if err != nil {
		return 0, err
	}
	if p <= 0 || p > 65535 {
		return 0, fmt.Errorf("invalid port: %d", p)
	}
	return p, nil
}

// lsdpQueryConn is the UDP surface used to broadcast LSDP queries.
// *net.UDPConn implements it.
type lsdpQueryConn interface {
	SetWriteDeadline(t time.Time) error
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
}

// lsdpSendHooks replaces the clock in tests. Zero values use real time.
type lsdpSendHooks struct {
	now          func() time.Time
	wait         func(context.Context, time.Duration) error
	jitterMillis func() int
}

func (h lsdpSendHooks) withDefaults() lsdpSendHooks {
	if h.now == nil {
		h.now = time.Now
	}
	if h.wait == nil {
		h.wait = waitLSDP
	}
	if h.jitterMillis == nil {
		h.jitterMillis = rand250ms
	}
	return h
}

func waitLSDP(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d < 0 {
		d = 0
	}
	timer := time.NewTimer(d)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func lsdpQueryOffsets() []time.Duration {
	return []time.Duration{
		0,
		time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		7 * time.Second,
		10 * time.Second,
	}
}

// sendLSDPQueries broadcasts query at 0s, 1s, 2s, 3s, 5s, 7s, and 10s from
// the start of the call. Each offset gets up to 249ms of jitter.
// The write deadline is refreshed before every send because SetReadDeadline
// does not restore an expired write deadline.
func sendLSDPQueries(ctx context.Context, conn lsdpQueryConn, port int, query []byte, dsts []net.IP, hooks lsdpSendHooks) {
	hooks = hooks.withDefaults()
	start := hooks.now()
	for _, offset := range lsdpQueryOffsets() {
		jitter := time.Duration(hooks.jitterMillis()) * time.Millisecond
		waitFor := offset + jitter - hooks.now().Sub(start)
		if err := hooks.wait(ctx, waitFor); err != nil {
			return
		}
		_ = conn.SetWriteDeadline(hooks.now().Add(2 * time.Second))
		for _, dst := range dsts {
			_, _ = conn.WriteToUDP(query, &net.UDPAddr{IP: dst, Port: port})
		}
	}
}

func rand250ms() int {
	// Small, deterministic-ish jitter. The value is a millisecond count.
	return int(time.Now().UnixNano() % int64(250*time.Millisecond/time.Millisecond))
}
