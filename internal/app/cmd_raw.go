package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

func cmdRaw(ctx context.Context, out *output.Printer, cfg config.Config, cache config.DiscoveryCache, deviceArg string, allowDiscover bool, discoverTimeout, httpTimeout time.Duration, dryRun bool, trace io.Writer, args []string) int {
	var (
		params   []string
		mutating bool
	)

	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--param":
			if i+1 >= len(args) {
				out.Errorf("raw: --param requires value")
				return 2
			}
			params = append(params, args[i+1])
			i++
		case strings.HasPrefix(args[i], "--param="):
			params = append(params, strings.TrimPrefix(args[i], "--param="))
		case args[i] == "--write":
			mutating = true
		default:
			rest = append(rest, args[i])
		}
	}

	if len(rest) == 0 {
		out.Errorf("raw: missing path (e.g. /Status)")
		return 2
	}
	if len(rest) > 1 {
		out.Errorf("raw: unexpected args: %q", strings.Join(rest[1:], " "))
		return 2
	}

	path := rest[0]
	if !strings.HasPrefix(path, "/") {
		out.Errorf("raw: path must start with '/' (got %q)", path)
		return 2
	}

	device, resolveErr := resolveDevice(ctx, cfg, cache, deviceArg, allowDiscover, discoverTimeout)
	if resolveErr != nil {
		out.Errorf("device: %v", resolveErr)
		return 1
	}

	query := map[string]string{}
	for _, p := range params {
		k, v, ok := strings.Cut(p, "=")
		if !ok || strings.TrimSpace(k) == "" {
			out.Errorf("raw: invalid --param %q (want k=v)", p)
			return 2
		}
		query[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: httpTimeout, DryRun: dryRun, Trace: trace})
	data, err := client.RawGet(ctx, path, query, mutating)
	if errors.Is(err, bluos.ErrDryRun) {
		return 0
	}
	if err != nil {
		out.Errorf("raw: %v", err)
		return 1
	}

	out.Print(map[string]any{
		"path": path,
		"xml":  string(data),
	})
	return 0
}
