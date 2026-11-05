package discovery

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func TestSendLSDPQueries_SevenAbsoluteOffsetsWithMillisecondJitter(t *testing.T) {
	// No t.Parallel: the fake conn models one socket's deadlines.

	const jitterMillis = 125
	jitter := time.Duration(jitterMillis) * time.Millisecond
	if jitter < time.Millisecond || jitter >= 250*time.Millisecond {
		t.Fatalf("jitter %s is not a millisecond count below 250ms", jitter)
	}

	offsets := lsdpQueryOffsets()
	dsts := []net.IP{net.IPv4(192, 168, 1, 255), net.IPv4(10, 0, 0, 255)}
	query := buildLSDPQueryAll()
	const port = 11430

	clock := &fakeClock{t: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	start := clock.now()
	conn := &deadlineUDP{now: clock.now}
	// Stale 2s write deadline, as SetDeadline used to arm it. A read-deadline
	// refresh must not clear it. WriteToUDP applies one before each packet.
	if err := conn.SetWriteDeadline(start.Add(2 * time.Second)); err != nil {
		t.Fatalf("set write deadline: %v", err)
	}
	_ = conn.SetReadDeadline(start.Add(150 * time.Millisecond))

	sendLSDPQueries(context.Background(), conn, port, query, dsts, lsdpSendHooks{
		now:          clock.now,
		wait:         clock.wait,
		jitterMillis: func() int { return jitterMillis },
	})

	wantSends := len(offsets) * len(dsts)
	ok := 0
	for _, attempt := range conn.writes {
		if attempt.err == nil {
			ok++
		}
	}
	if ok != wantSends {
		t.Fatalf("successful sends = %d, want %d\n%s", ok, wantSends, formatLSDPAttempts(start, conn.writes))
	}
	for i, offset := range offsets {
		wantAt := start.Add(offset + jitter)
		for j, dst := range dsts {
			got := conn.writes[i*len(dsts)+j]
			if got.err != nil {
				t.Fatalf("send %d dest %d: %v\n%s", i, j, got.err, formatLSDPAttempts(start, conn.writes))
			}
			if !got.at.Equal(wantAt) {
				t.Fatalf("send %d dest %d at %s, want %s (documented offset plus millisecond jitter)\n%s",
					i, j, got.at.Sub(start), wantAt.Sub(start), formatLSDPAttempts(start, conn.writes))
			}
			if got.port != port || !got.ip.Equal(dst) || !bytes.Equal(got.payload, query) {
				t.Fatalf("send %d dest %d = %s:%d %q, want %s:%d query", i, j, got.ip, got.port, got.payload, dst, port)
			}
		}
	}
}

func TestWaitLSDP_ReturnsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitLSDP(ctx, time.Hour); err == nil {
		t.Fatal("waitLSDP returned nil after cancel")
	}
}

type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time { return c.t }

func (c *fakeClock) wait(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d > 0 {
		c.t = c.t.Add(d)
	}
	return nil
}

type udpAttempt struct {
	at      time.Time
	ip      net.IP
	port    int
	payload []byte
	err     error
}

// deadlineUDP keeps the write deadline until SetWriteDeadline replaces it.
// SetReadDeadline does not restore writes, matching net.UDPConn.
type deadlineUDP struct {
	now     func() time.Time
	writeDL time.Time
	writes  []udpAttempt
}

func (c *deadlineUDP) SetWriteDeadline(t time.Time) error {
	c.writeDL = t
	return nil
}

func (c *deadlineUDP) SetReadDeadline(time.Time) error {
	return nil
}

func (c *deadlineUDP) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	// Interleave the read loop's deadline refresh. It must not extend writes.
	_ = c.SetReadDeadline(c.now().Add(150 * time.Millisecond))
	now := c.now()
	attempt := udpAttempt{at: now, payload: append([]byte(nil), b...)}
	if addr != nil {
		attempt.ip = append(net.IP(nil), addr.IP...)
		attempt.port = addr.Port
	}
	if !c.writeDL.IsZero() && now.After(c.writeDL) {
		attempt.err = &net.OpError{Op: "write", Net: "udp", Err: os.ErrDeadlineExceeded}
		c.writes = append(c.writes, attempt)
		return 0, attempt.err
	}
	c.writes = append(c.writes, attempt)
	return len(b), nil
}

func formatLSDPAttempts(start time.Time, writes []udpAttempt) string {
	var buf bytes.Buffer
	for i, w := range writes {
		status := "ok"
		if w.err != nil {
			status = w.err.Error()
		}
		fmt.Fprintf(&buf, "%d: %s %s %s\n", i, w.at.Sub(start), w.ip, status)
	}
	return buf.String()
}
