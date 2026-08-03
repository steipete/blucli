# blucli 🫐 — Pick a player, press play.

[![CI](https://img.shields.io/github/actions/workflow/status/steipete/blucli/ci.yml?branch=main&style=flat-square&label=ci)](https://github.com/steipete/blucli/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/v/release/steipete/blucli?style=flat-square)](https://github.com/steipete/blucli/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/github/license/steipete/blucli?style=flat-square)](LICENSE)
[![Homebrew](https://img.shields.io/badge/Homebrew-steipete%2Ftap-FBB040?style=flat-square&logo=homebrew&logoColor=black)](https://github.com/steipete/homebrew-tap)

`blucli` is a command-line client for discovering and controlling Bluesound and NAD players that run BluOS. It gives people and scripts access to playback, grouping, queues, presets, browsing, and diagnostics over the local network.

## Install

Homebrew installs the `blu` command on macOS or Linux:

```bash
brew install steipete/tap/blucli
```

With Go 1.25 or newer:

```bash
go install github.com/steipete/blucli/cmd/blu@latest
```

Prebuilt macOS, Linux, and Windows archives are available from [GitHub Releases](https://github.com/steipete/blucli/releases/latest). For a container-based setup, see the [Docker guide](docs/usage.md#docker).

## Quick start

Discover players, then address one by its discovery name or `host:port`:

```bash
blu devices
blu --device "Living Room" status
blu --device 192.168.1.19:11000 now
```

When discovery finds exactly one player, `blu status` selects it automatically. Otherwise, pass `--device`, set `BLU_DEVICE`, or configure a default.

## Commands

| Area | Commands |
| --- | --- |
| Discovery and status | `devices`, `status`, `now`, `watch status\|sync` |
| Playback | `play`, `pause`, `stop`, `next`, `prev`, `shuffle`, `repeat`, `sleep` |
| Volume and groups | `volume`, `mute`, `group` |
| Content | `queue`, `presets`, `browse`, `playlists`, `inputs`, `tunein` |
| Spotify | `spotify open`, plus optional Web API login, search, and playback |
| Automation and diagnostics | `--json`, `--dry-run`, `--trace-http`, `diag`, `doctor`, `raw` |

See the [usage guide](docs/usage.md) for command examples, Spotify setup, Docker, shell completions, and automation notes. The [protocol and implementation spec](docs/spec.md) documents discovery, BluOS endpoints, CLI behavior, and the project layout.

## Device selection

`blu` resolves a target in this order:

1. `--device <host:port|name|alias>`
2. `BLU_DEVICE`
3. `default_device` in the config file
4. the discovery cache, when it contains exactly one player
5. live discovery, when it finds exactly one player

Run `blu devices` to refresh the cache. Discovery names work directly; aliases are useful for custom shortcuts or disambiguation.

## Configuration

The default config file is `~/Library/Application Support/blu/config.json` on macOS and `~/.config/blu/config.json` on Linux. Override it with `--config <path>`.

```json
{
  "default_device": "192.168.1.19:11000",
  "aliases": {
    "kitchen": "192.168.1.19:11000",
    "office": "192.168.1.115:11000"
  }
}
```

## Development

```bash
go build -o dist/blu ./cmd/blu
go test ./...
golangci-lint run --timeout=5m
```

The repository also provides equivalent `pnpm build`, `pnpm test`, and `pnpm lint` helpers.

## License

[MIT](LICENSE)
