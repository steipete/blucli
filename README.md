# blucli

BluOS CLI (`blu`) for Bluesound/NAD BluOS players.

Spec: `docs/spec.md`

## Install

```bash
go install github.com/steipete/blucli/cmd/blu@latest
```

Or grab a prebuilt binary from GitHub Releases.

## Usage

```bash
blu version
blu devices
blu --device kitchen status
blu --json status
blu play
blu volume set 15
blu --dry-run --trace-http queue clear
blu group add office --name "Downstairs"
```

## Shell completions

```bash
source <(blu completions bash)
```

## Prior work / references

- BluShell (PowerShell wrapper + unofficial docs): https://github.com/albertony/blushell (no license file in repo as of 2025-12-17)
- pyblu (Python library): https://github.com/LouisChrist/pyblu (MIT)
- BluOS Controller.app (macOS): Electron app; inspect `app.asar` for discovery details
