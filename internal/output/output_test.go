package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/discovery"
)

func TestPrinter_Print_Text(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	p := New(Options{Stdout: &out, Stderr: &errOut})

	p.Print([]discovery.Device{})
	if got := out.String(); !strings.Contains(got, "no devices") {
		t.Fatalf("stdout = %q; want contains no devices", got)
	}
	out.Reset()

	p.Print([]discovery.Device{{ID: "d1", Type: "Player", Version: "1.2.3"}})
	if got := out.String(); !strings.Contains(got, "d1 (Player) v1.2.3") {
		t.Fatalf("stdout = %q; want contains device line", got)
	}
	out.Reset()

	p.Print(bluos.Status{State: "play", Volume: 15, Mute: false, Artist: "A", Title: "T"})
	if got := out.String(); !strings.Contains(got, "vol=15") || !strings.Contains(got, "A — T") {
		t.Fatalf("stdout = %q; want formatted status", got)
	}
	out.Reset()

	p.Print(bluos.SyncStatus{})
	if got := out.String(); !strings.Contains(got, "no group") {
		t.Fatalf("stdout = %q; want contains no group", got)
	}
	out.Reset()

	p.Print(bluos.SyncStatus{
		Group:  "G",
		Master: &bluos.SyncMaster{Host: "host", Port: 11000},
		Slaves: []bluos.SyncSlave{{ID: "s1", Port: 11001}},
	})
	if got := out.String(); !strings.Contains(got, "group: G") || !strings.Contains(got, "master: host:11000") || !strings.Contains(got, "s1:11001") {
		t.Fatalf("stdout = %q; want formatted syncstatus", got)
	}
	out.Reset()

	p.Print(bluos.Playlist{ID: 1, Length: 1, Songs: []bluos.PlaylistSong{{ID: 7, Artist: "AR", Fn: "file.mp3"}}})
	if got := out.String(); !strings.Contains(got, "(queue)") || !strings.Contains(got, "AR — file.mp3") {
		t.Fatalf("stdout = %q; want formatted playlist", got)
	}
	out.Reset()

	p.Print(bluos.Presets{})
	if got := out.String(); !strings.Contains(got, "no presets") {
		t.Fatalf("stdout = %q; want contains no presets", got)
	}
	out.Reset()

	p.Print(bluos.Browse{Items: []bluos.BrowseItem{{Text: "X", BrowseKey: "k1"}, {Text: "Y", PlayURL: "p1"}, {Type: "noop"}}})
	if got := out.String(); !strings.Contains(got, "[k1]") || !strings.Contains(got, "(p1)") {
		t.Fatalf("stdout = %q; want browse formatting", got)
	}
	out.Reset()

	p.Print(bluos.Playlists{})
	if got := out.String(); !strings.Contains(got, "no playlists") {
		t.Fatalf("stdout = %q; want contains no playlists", got)
	}
	out.Reset()

	p.Print(bluos.RadioBrowse{})
	if got := out.String(); !strings.Contains(got, "no inputs") {
		t.Fatalf("stdout = %q; want contains no inputs", got)
	}
	out.Reset()

	p.Print(bluos.RadioBrowse{Items: []bluos.RadioItem{{ID: "i1", Text: "Input"}}})
	if got := out.String(); !strings.Contains(got, "i1  Input") {
		t.Fatalf("stdout = %q; want formatted inputs", got)
	}
	out.Reset()

	p.Print(bluos.RadioBrowse{Categories: []bluos.RadioCategory{{Text: "Stations", Items: []bluos.RadioItem{{ID: "s1", Text: "Foo"}}}}})
	if got := out.String(); !strings.Contains(got, "Stations:") || !strings.Contains(got, "s1  Foo") {
		t.Fatalf("stdout = %q; want formatted categories", got)
	}
}

func TestPrinter_Print_JSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	p := New(Options{JSON: true, Stdout: &out, Stderr: &bytes.Buffer{}})
	p.Print(map[string]any{"ok": true, "n": 1})
	if got := out.String(); !strings.Contains(got, "\"ok\": true") {
		t.Fatalf("stdout = %q; want json output", got)
	}
}

func TestParseIntInRange(t *testing.T) {
	t.Parallel()

	if _, err := ParseIntInRange("", 0, 10); err == nil {
		t.Fatalf("want error for empty")
	}
	if _, err := ParseIntInRange("nope", 0, 10); err == nil {
		t.Fatalf("want error for invalid")
	}
	if _, err := ParseIntInRange("11", 0, 10); err == nil {
		t.Fatalf("want error for out of range")
	}
	if got, err := ParseIntInRange(" 7 ", 0, 10); err != nil || got != 7 {
		t.Fatalf("n=%d err=%v; want 7 nil", got, err)
	}
}
