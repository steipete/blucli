# Usage guide

This guide collects command examples and optional integrations for `blu`. Run `blu --help` for the command summary and see the [protocol spec](spec.md) for endpoint and implementation details.

## Playback

```bash
blu status
blu now
blu play
blu pause
blu stop
blu next
blu prev
blu play --url https://samplelib.com/lib/preview/mp3/sample-3s.mp3
```

## TuneIn

Search or start a station without configuring a separate streaming API:

```bash
blu tunein search "Gareth Emery"
blu tunein play "Gareth Emery"
blu tunein play --pick 0 "Gareth Emery"
```

## Volume, playback modes, and groups

```bash
blu volume get
blu volume set 15
blu volume up
blu volume down
blu mute on
blu mute off
blu mute toggle
blu shuffle on
blu shuffle off
blu repeat off
blu repeat track
blu repeat queue

blu group status
blu group add 192.168.1.115:11000 --name "Downstairs"
blu group remove 192.168.1.115:11000
```

## Queue, presets, and browsing

```bash
blu queue list
blu queue clear
blu presets list
blu presets load 1
blu browse --key "TuneIn:"
blu playlists
blu inputs
```

## Automation and diagnostics

`--json` emits stable machine output. `--dry-run` permits reads, blocks mutating requests, and prints request URLs; `--trace-http` prints request URLs without blocking writes.

```bash
blu --json status
blu diag
blu doctor
blu raw /Status
blu --dry-run --trace-http raw /Play --param url=https://samplelib.com/lib/preview/mp3/sample-3s.mp3 --write
```

## Spotify

BluOS exposes Spotify Connect directly:

```bash
blu spotify open
```

Search and playback through the Spotify Web API require a Spotify developer app. Add `http://127.0.0.1:8974/callback` to the app's redirect URLs, set `SPOTIFY_CLIENT_ID` or pass `--client-id`, then log in:

```bash
blu spotify login
blu spotify search "Gareth Emery"
blu spotify play "Gareth Emery"
```

As an alternative, save a BluOS preset from Spotify and load it with `blu presets load <id>`.

## Shell completions

Generate completion scripts for Bash or Zsh:

```bash
source <(blu completions bash)
```

```zsh
source <(blu completions zsh)
```

## Docker

Build the image locally:

```bash
docker build -t blucli .
```

Persist configuration and discovery data under `.blu`:

```bash
docker run --rm --network host -v "$PWD/.blu:/data" blucli devices
docker run --rm --network host -v "$PWD/.blu:/data" blucli --device 192.168.1.19:11000 status
```

Linux containers need host networking for discovery. On other setups, pass an explicit `--device` or set `BLU_DEVICE`.

## Source helpers

The package scripts wrap the Go build and quality checks:

```bash
go run ./cmd/blu --help
pnpm build
pnpm test
pnpm lint
pnpm format
pnpm blu -- status
```

## Prior work and references

- [BluShell](https://github.com/albertony/blushell) is a PowerShell wrapper with unofficial BluOS documentation.
- [pyblu](https://github.com/LouisChrist/pyblu) is an MIT-licensed Python library for BluOS.
- The BluOS Controller app informed the discovery comparison in the [protocol spec](spec.md#reference-comparison-notes).
