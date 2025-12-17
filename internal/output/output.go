package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/discovery"
)

type Options struct {
	JSON   bool
	Stdout io.Writer
	Stderr io.Writer
}

type Printer struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
}

func New(opts Options) *Printer {
	return &Printer{
		json:   opts.JSON,
		stdout: opts.Stdout,
		stderr: opts.Stderr,
	}
}

func (p *Printer) Stdout() io.Writer { return p.stdout }
func (p *Printer) Stderr() io.Writer { return p.stderr }

func (p *Printer) Print(v any) {
	if p.json {
		p.printJSON(v)
		return
	}

	switch value := v.(type) {
	case []discovery.Device:
		p.printDevices(value)
	case bluos.Status:
		p.printStatus(value)
	case bluos.SyncStatus:
		p.printSyncStatus(value)
	case bluos.Playlist:
		p.printPlaylist(value)
	case bluos.Presets:
		p.printPresets(value)
	case bluos.Browse:
		p.printBrowse(value)
	case bluos.Playlists:
		p.printPlaylists(value)
	case bluos.RadioBrowse:
		p.printRadioBrowse(value)
	case map[string]any:
		p.printJSON(value)
	default:
		p.printJSON(v)
	}
}

func (p *Printer) Errorf(format string, args ...any) {
	fmt.Fprintf(p.stderr, format+"\n", args...)
}

func (p *Printer) Warnf(format string, args ...any) {
	fmt.Fprintf(p.stderr, "warn: "+format+"\n", args...)
}

func (p *Printer) printJSON(v any) {
	enc := json.NewEncoder(p.stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (p *Printer) printDevices(devices []discovery.Device) {
	if len(devices) == 0 {
		fmt.Fprintln(p.stdout, "no devices")
		return
	}
	for _, d := range devices {
		extra := ""
		if d.Version != "" {
			extra = " v" + d.Version
		}
		fmt.Fprintf(p.stdout, "%s (%s)%s\n", d.ID, d.Type, extra)
	}
}

func (p *Printer) printStatus(s bluos.Status) {
	line := strings.TrimSpace(fmt.Sprintf("%s | vol=%d mute=%t | %s — %s",
		strings.TrimSpace(s.State),
		s.Volume,
		s.Mute,
		strings.TrimSpace(s.Artist),
		strings.TrimSpace(s.Title),
	))
	fmt.Fprintln(p.stdout, line)
}

func (p *Printer) printSyncStatus(s bluos.SyncStatus) {
	if s.Group == "" && s.Master == nil && len(s.Slaves) == 0 {
		fmt.Fprintln(p.stdout, "no group")
		return
	}
	fmt.Fprintf(p.stdout, "group: %s\n", strings.TrimSpace(s.Group))
	if s.Master != nil {
		fmt.Fprintf(p.stdout, "master: %s:%d\n", strings.TrimSpace(s.Master.Host), s.Master.Port)
	} else {
		fmt.Fprintln(p.stdout, "master: (this player)")
	}
	if len(s.Slaves) == 0 {
		fmt.Fprintln(p.stdout, "slaves: none")
		return
	}
	fmt.Fprintln(p.stdout, "slaves:")
	for _, slave := range s.Slaves {
		fmt.Fprintf(p.stdout, "  - %s:%d\n", slave.ID, slave.Port)
	}
}

func (p *Printer) printPlaylist(pl bluos.Playlist) {
	name := strings.TrimSpace(pl.Name)
	if name == "" {
		name = "(queue)"
	}
	fmt.Fprintf(p.stdout, "%s id=%d len=%d modified=%d\n", name, pl.ID, pl.Length, pl.Modified)
	for _, song := range pl.Songs {
		title := strings.TrimSpace(song.Title)
		artist := strings.TrimSpace(song.Artist)
		if title == "" {
			title = strings.TrimSpace(song.Fn)
		}
		fmt.Fprintf(p.stdout, "%3d  %s — %s\n", song.ID, artist, title)
	}
}

func (p *Printer) printPresets(pr bluos.Presets) {
	if len(pr.Presets) == 0 {
		fmt.Fprintln(p.stdout, "no presets")
		return
	}
	for _, preset := range pr.Presets {
		fmt.Fprintf(p.stdout, "%2d  %s\n", preset.ID, strings.TrimSpace(preset.Name))
	}
}

func (p *Printer) printBrowse(b bluos.Browse) {
	if len(b.Items) == 0 {
		fmt.Fprintln(p.stdout, "no items")
		return
	}
	for i, item := range b.Items {
		label := strings.TrimSpace(item.Text)
		if label == "" {
			label = item.Type
		}
		key := strings.TrimSpace(item.BrowseKey)
		play := strings.TrimSpace(item.PlayURL)
		switch {
		case key != "":
			fmt.Fprintf(p.stdout, "%3d  %s  [%s]\n", i, label, key)
		case play != "":
			fmt.Fprintf(p.stdout, "%3d  %s  (%s)\n", i, label, play)
		default:
			fmt.Fprintf(p.stdout, "%3d  %s\n", i, label)
		}
	}
}

func (p *Printer) printPlaylists(pl bluos.Playlists) {
	if len(pl.Names) == 0 {
		fmt.Fprintln(p.stdout, "no playlists")
		return
	}
	for _, n := range pl.Names {
		fmt.Fprintln(p.stdout, strings.TrimSpace(n.Text))
	}
}

func (p *Printer) printRadioBrowse(rb bluos.RadioBrowse) {
	if len(rb.Items) == 0 {
		fmt.Fprintln(p.stdout, "no inputs")
		return
	}
	for _, item := range rb.Items {
		fmt.Fprintf(p.stdout, "%s  %s\n", item.ID, strings.TrimSpace(item.Text))
	}
}

func ParseIntInRange(s string, min, max int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("out of range (%d..%d): %d", min, max, n)
	}
	return n, nil
}
