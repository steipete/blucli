package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/steipete/blucli/internal/bluos"
	"github.com/steipete/blucli/internal/config"
	"github.com/steipete/blucli/internal/output"
)

const (
	defaultDiscoveryTimeout = 4 * time.Second
	defaultHTTPTimeout      = 5 * time.Second
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	global := flag.NewFlagSet("blu", flag.ContinueOnError)
	global.SetOutput(stderr)

	var (
		flagDevice     = global.String("device", "", "device id/alias (host[:port] or alias)")
		flagJSON       = global.Bool("json", false, "json output")
		flagTimeout    = global.Duration("timeout", defaultHTTPTimeout, "http timeout")
		flagDryRun     = global.Bool("dry-run", false, "log requests; block mutating requests")
		flagTraceHTTP  = global.Bool("trace-http", false, "log HTTP requests to stderr")
		flagHelp       = global.Bool("help", false, "print help")
		flagH          = global.Bool("h", false, "print help")
		flagVersion    = global.Bool("version", false, "print version")
		flagV          = global.Bool("v", false, "print version")
		flagDiscover   = global.Bool("discover", true, "allow discovery when needed")
		flagDiscTO     = global.Duration("discover-timeout", defaultDiscoveryTimeout, "discovery timeout")
		flagConfigPath = global.String("config", "", "config path (optional)")
	)

	if err := global.Parse(args); err != nil {
		return 2
	}

	if *flagHelp || *flagH {
		usage(stdout)
		return 0
	}

	if *flagVersion || *flagV {
		fmt.Fprintln(stdout, Version)
		return 0
	}

	cmdArgs := global.Args()
	if len(cmdArgs) == 0 {
		usage(stderr)
		return 2
	}

	cfg, err := config.Load(config.LoadOptions{Path: *flagConfigPath})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}

	paths, err := config.Paths()
	if err != nil {
		fmt.Fprintf(stderr, "config paths: %v\n", err)
		return 1
	}

	cache, err := config.LoadDiscoveryCache(paths.CachePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "cache: %v\n", err)
		return 1
	}

	out := output.New(output.Options{
		JSON:   *flagJSON,
		Stdout: stdout,
		Stderr: stderr,
	})

	switch cmdArgs[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	case "version":
		fmt.Fprintln(stdout, Version)
		return 0
	case "completions":
		return cmdCompletions(out, cmdArgs[1:])
	case "devices":
		return cmdDevices(ctx, out, paths, cfg, cache, *flagDiscTO)
	case "status":
		device, resolveErr := resolveDevice(ctx, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO)
		if resolveErr != nil {
			out.Errorf("device: %v", resolveErr)
			return 1
		}
		client := bluos.NewClient(device.BaseURL(), bluos.Options{Timeout: *flagTimeout, DryRun: *flagDryRun, Trace: traceWriter(*flagTraceHTTP, *flagDryRun, stderr)})
		status, err := client.Status(ctx, bluos.StatusOptions{})
		if err != nil {
			out.Errorf("status: %v", err)
			return 1
		}
		out.Print(status)
		return 0
	case "now":
		return cmdNow(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr))
	case "watch":
		return cmdWatch(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "play", "pause", "stop", "next", "prev":
		return cmdPlayback(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[0], cmdArgs[1:])
	case "shuffle":
		return cmdShuffle(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "repeat":
		return cmdRepeat(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "volume":
		return cmdVolume(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "mute":
		return cmdMute(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "group":
		return cmdGroup(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "queue":
		return cmdQueue(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "presets":
		return cmdPresets(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "browse":
		return cmdBrowse(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "playlists":
		return cmdPlaylists(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "inputs":
		return cmdInputs(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	case "sleep":
		return cmdSleep(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr))
	case "diag":
		return cmdDiag(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr))
	case "doctor":
		return cmdDoctor(ctx, out, cfg, cache, *flagDiscTO, *flagTimeout)
	case "raw":
		return cmdRaw(ctx, out, cfg, cache, *flagDevice, *flagDiscover, *flagDiscTO, *flagTimeout, *flagDryRun, traceWriter(*flagTraceHTTP, *flagDryRun, stderr), cmdArgs[1:])
	default:
		out.Errorf("unknown command: %q", cmdArgs[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "blu — BluOS CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  blu [--help] [--version] [--device <id|alias>] [--json] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  version")
	fmt.Fprintln(w, "  completions bash|zsh")
	fmt.Fprintln(w, "  devices")
	fmt.Fprintln(w, "  status|now")
	fmt.Fprintln(w, "  watch status|sync")
	fmt.Fprintln(w, "  play [--url <url>] [--seek <seconds>] [--id <n>]")
	fmt.Fprintln(w, "  pause|stop|next|prev")
	fmt.Fprintln(w, "  shuffle on|off")
	fmt.Fprintln(w, "  repeat off|track|queue")
	fmt.Fprintln(w, "  volume get|set <0-100>|up|down")
	fmt.Fprintln(w, "  mute on|off|toggle")
	fmt.Fprintln(w, "  group status|add <slave> [--name <group>]|remove <slave>")
	fmt.Fprintln(w, "  queue list|clear|delete <id>|move <old> <new>|save <name>")
	fmt.Fprintln(w, "  presets list|load <id|+1|-1>")
	fmt.Fprintln(w, "  browse --key <key> [--q <query>] [--context]")
	fmt.Fprintln(w, "  playlists [--service <name>] [--category <cat>] [--expr <search>]")
	fmt.Fprintln(w, "  inputs [play <id>]")
	fmt.Fprintln(w, "  sleep")
	fmt.Fprintln(w, "  diag|doctor")
	fmt.Fprintln(w, "  raw <path> [--param k=v ...] [--write]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Env:")
	fmt.Fprintln(w, "  BLU_DEVICE  default device id/alias")
}

func traceWriter(traceHTTP, dryRun bool, stderr io.Writer) io.Writer {
	if !traceHTTP && !dryRun {
		return nil
	}
	return stderr
}
