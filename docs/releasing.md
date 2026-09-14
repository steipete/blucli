# Releasing blucli

Releases use `.github/workflows/release-unified.yml`, pinned to
`openclaw/release-workflows` v1.9.0 at
`f613cbfed2b043159c850c353e7facb8c89833b0`. The shared Go CLI archetype
builds with GoReleaser, signs and notarizes the macOS binaries, and verifies
an immutable artifact independently on Intel and Apple Silicon before publishing.

## Release contract

- macOS 12.0 or newer; both `darwin_amd64` and `darwin_arm64`.
- Developer ID Application: Peter Steinberger (Y5PE65HELJ), with hardened runtime,
  trusted timestamp, and signing identifier `com.steipete.blucli.blu`.
- The executable remains `blu` (`blu.exe` on Windows).
- Archives remain `blucli_<version>_<os>_<arch>.tar.gz` (Windows: `.zip`),
  including `LICENSE` and `README.md`. There is no universal archive.
- `checksums.txt` covers the published payload and provenance controls:
  `ASSET-INVENTORY.json`, `SIGNING-MANIFEST.json`, and `RELEASE-NOTES.md`.
- The handoff updates `steipete/homebrew-tap` formula `blucli` with the exact
  verified asset names and SHA-256 values, then verifies the resulting formula.

The release builds on macOS so the GoReleaser post-build hook can inspect every
Darwin binary with `otool -arch all -l`. `CGO_ENABLED=0` and
`MACOSX_DEPLOYMENT_TARGET=12.0` are explicit; the hook rejects a deployment target
other than 12.0, including toolchain-default changes. If cgo is introduced, also
set matching `CGO_CFLAGS` and `CGO_LDFLAGS` with `-mmacosx-version-min=12.0`;
the environment variable alone does not reliably constrain external linking.
CI exercises the same build configuration and target gate without Apple secrets.

## Repository setup

Protect `main` with required CI checks. Enable Actions read/write workflow
permissions and `can_approve_pull_request_reviews` so closeout can create its PR.
All workflows still declare their own limited permissions.

The caller maps these repository secrets to the shared workflow:

| Repository secret | Shared workflow secret |
| --- | --- |
| `MACOS_SIGN_P12` | `MACOS_SIGNING_P12` |
| `MACOS_SIGN_P12_PASSWORD` | `MACOS_SIGNING_P12_PASSWORD` |
| `ASC_KEY_ID` | `ASC_KEY_ID` |
| `ASC_ISSUER_ID` | `ASC_ISSUER_ID` |
| `ASC_PRIVATE_KEY` | `ASC_PRIVATE_KEY_P8` |
| `HOMEBREW_TAP_TOKEN` | `TAP_TOKEN` |

The tap token needs Contents read and Actions write on `steipete/homebrew-tap`.
The signing job checks the personal identity and cleans up its temporary keychain.
The independent verifiers do not receive signing or release-write credentials.

## Ship a patch

1. Check the current tags and published releases. Finalize the versioned
   Unreleased changelog section with a date and Highlights; update the source
   version in `cmd/blu/main.go` to the new version. GoReleaser overrides it with
   the release tag, while source installs report the checked-in version.
2. Run `actionlint`, `go test -race ./...`, `golangci-lint run --timeout=5m`,
   `python3 -B -m unittest discover -s scripts -p 'test_*.py'`, and
   `goreleaser build --snapshot --clean` on macOS. Review and merge the PR, then
   wait for CI on the exact `main` commit to pass.
3. Recheck that the new tag/release does not exist. Dispatch from current `main`:

   ```sh
   gh workflow run release-unified.yml --ref main -f version=0.1.7
   ```

   The workflow creates the annotated tag at the frozen protected commit. Do not
   create or move the tag manually. Retries reuse that tag; investigate failures
   before rerunning. The release body is the exact dated changelog section.
4. Verify the release assets, checksums, macOS signatures and execution, Go proxy,
   and Homebrew formula. Review and merge the generated closeout PR, naming the
   next section `## <next-patch> (Unreleased)` to match this repository.

## Verify a download

Download an archive and `checksums.txt` from the same release. Verify its SHA-256
before extracting it. For example, for an Apple Silicon download:

```sh
shasum -a 256 blucli_0.1.7_darwin_arm64.tar.gz
# Compare with that exact filename in checksums.txt.
tar -xzf blucli_0.1.7_darwin_arm64.tar.gz
codesign -dvv ./blu
codesign --verify --deep --strict --verbose=4 ./blu
codesign --verify --strict --check-notarization -R=notarized ./blu
spctl -a -vv -t open --context context:primary-signature ./blu
python3 scripts/check_macos_target.py ./blu
./blu --version
```

`codesign` must report the identity and Team ID above; Gatekeeper must accept it.
Use `otool -l ./blu` outside a source checkout and inspect `LC_BUILD_VERSION`:
`minos` must be `12.0`. Test a quarantined download without removing its quarantine
attribute. Command-line `curl` does not normally add quarantine on macOS, so a
manual Gatekeeper test must explicitly add `com.apple.quarantine` before launch.
