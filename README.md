# 🫐 blucli — play, group, and automate BluOS.

BluOS CLI (`blu`) for Bluesound/NAD BluOS players.

Spec: [docs/spec.md](docs/spec.md)

## Install / Run

- Homebrew (installs `blu`): `brew install steipete/tap/blucli`
- Go install: `go install github.com/steipete/blucli/cmd/blu@latest`
- From source: `go run ./cmd/blu --help`

## Features

- Discovery: mDNS (`_musc/_musp/_musz/_mush`) + LSDP fallback (`blu devices`)
- Device selection: `--device`, `BLU_DEVICE`, config `default_device`, aliases, discovery names
- Playback: `play/pause/stop/next/prev` + `play --url/--seek/--id`
- Volume + modes: `volume …`, `mute …`, `shuffle …`, `repeat …`
- Grouping: `group status|add|remove`
- Queue/presets/browse: `queue …`, `presets …`, `browse …`, `playlists …`, `inputs …`
- TuneIn: `tunein search|play` for quick “play X”
- Spotify Connect: `spotify open` (and optional Web API `spotify login/search/play`)
- Sleep timer: `sleep`
- Watch: long-poll `Status` / `SyncStatus` (`watch status|sync`)
- Scripting/safety: `--json`, `--dry-run`, `--trace-http`
- Diagnostics: `diag`, `doctor`, `raw` endpoint runner
- Shell completions: `completions bash|zsh`

## Quickstart

```bash
blu devices
blu --device 192.168.1.19:11000 status
```

## Device selection

`blu` picks a target device in this order:

1. `--device <id|name|alias>` (e.g. `192.168.1.19:11000` or `Schlafzimmer`)
2. `BLU_DEVICE`
3. config `default_device`
4. discovery cache / live discovery (only if exactly 1 device)

If multiple devices exist, run `blu devices` and pick one.

## Config (aliases)

You can also target a player by its discovery name (shown by `blu devices`) without writing a config file. Aliases are just for custom shortcuts / disambiguation.

Config file:
- macOS: `~/Library/Application Support/blu/config.json`
- Linux: `~/.config/blu/config.json`

Example:

```json
{
  "default_device": "192.168.1.19:11000",
  "aliases": {
    "kitchen": "192.168.1.19:11000",
    "office": "192.168.1.115:11000"
  }
}
```

## Common commands

Playback:

```bash
blu status
blu now
blu play
blu pause
blu stop
blu next
blu prev

# Play a stream URL
blu play --url http://ice1.somafm.com/groovesalad-128-mp3
```

“Say a thing, play something” (TuneIn-backed):

```bash
blu tunein search "Gareth Emery"
blu tunein play "Gareth Emery"
blu tunein play --pick 0 "Gareth Emery"
```

Volume / repeat / shuffle:

```bash
blu volume get
blu volume set 15
blu volume up
blu volume down
blu mute on|off|toggle
blu shuffle on|off
blu repeat off|track|queue
```

Grouping:

```bash
blu group status
blu group add 192.168.1.115:11000 --name "Downstairs"
blu group remove 192.168.1.115:11000
```

Queue / presets / browse:

```bash
blu queue list
blu presets list
blu browse --key "TuneIn:"
blu inputs
```

Diagnostics:

```bash
blu diag
blu doctor
```

Power user:

```bash
blu raw /Status
blu --dry-run --trace-http raw /Play --param url=http://ice1.somafm.com/groovesalad-128-mp3 --write
```

## Scripting + safety

- `--json`: stable machine output.
- `--dry-run`: blocks mutating requests but still allows reads; always logs request URLs.
- `--trace-http`: also logs request URLs (useful without `--dry-run`).

## Shell completions

```bash
source <(blu completions bash)
```

## Spotify notes

BluOS uses Spotify Connect.

Instant “switch player into Spotify”:

```bash
blu spotify open
```

Optional: Spotify Web API integration (OAuth) for `blu spotify search` / `blu spotify play`:

1. Create a Spotify developer app and add a redirect URL (default): `http://127.0.0.1:8974/callback`
2. Set `SPOTIFY_CLIENT_ID` (or pass `--client-id`)
3. Login:

```bash
blu spotify login
```

Then:

```bash
blu spotify play "Gareth Emery"
```

Fallback: save a BluOS preset from Spotify, then `blu presets load <id>`.

## Development

Go-only:

```bash
go test ./...
golangci-lint run --timeout=5m
```

Convenience scripts (optional):

```bash
pnpm build
pnpm test
pnpm lint
pnpm format
pnpm blu -- status
```

## Prior work / references

- BluShell (PowerShell wrapper + unofficial docs): https://github.com/albertony/blushell (no license file in repo as of 2025-12-17)
- pyblu (Python library): https://github.com/LouisChrist/pyblu (MIT)
- BluOS Controller.app (macOS): Electron app; inspect `app.asar` for discovery details
