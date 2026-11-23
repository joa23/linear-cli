# Releasing and verifying linear-cli

Relynk distributes one static binary per platform from the GitHub Releases of
[`Relynk/linear-cli`](https://github.com/Relynk/linear-cli). Each release is:

- **built only in CI**, by `.github/workflows/release.yml`, from a `vX.Y.Z` tag;
- **locked**: the Go toolchain is exactly the one in `go.mod`
  (`GOTOOLCHAIN=local`), dependencies come from `go.sum` and are checked with
  `go mod verify` (nothing is upgraded or tidied at release time), and every
  GitHub Action and GoReleaser itself is pinned to an exact version/commit;
- **static and reproducible**: `CGO_ENABLED=0`, `-trimpath`, commit-derived
  timestamps, so the same tag gives byte-identical binaries and archives;
- **verifiable**: `checksums.txt` (SHA-256) plus a GitHub build provenance
  attestation (Sigstore, signed by the release workflow) for each binary,
  archive and `checksums.txt`.

## Cutting a release

1. Make sure `main` is green in CI (`.github/workflows/ci.yml`).
2. Update `CHANGELOG.md`, commit, and push.
3. Tag and push the tag (use a version above the upstream `v1.11.0`, so
   Relynk builds never collide with upstream tags):
   ```bash
   git tag -s vX.Y.Z -m "vX.Y.Z"    # -s: signed tag, recommended
   git push origin vX.Y.Z
   ```
4. The workflow then
   1. runs `go mod verify`, `go mod tidy -diff`, `go vet` and `go test` with a
      read-only token;
   2. builds all platforms with GoReleaser and uploads them to a **draft**
      release;
   3. creates the build provenance attestation;
   4. publishes the draft. If any step fails, the release stays a draft.
5. Update `Formula/linear-cli.rb` (version + four `sha256` values) from
   `gh release download vX.Y.Z --repo Relynk/linear-cli --pattern checksums.txt --output -`.

Never run `goreleaser release` locally; CI is the only source of release
artifacts.

Recommended repository settings (admin):
- a tag ruleset protecting `v*` (only maintainers can create, nobody can
  move or delete tags);
- the `release` environment (created on first run) with required reviewers
  and "protected tags only";
- **immutable releases** enabled, so published assets and tags cannot be
  replaced (the draft-then-publish flow above is compatible with it).

Note: artifact attestations on a **private** repository require GitHub
Enterprise Cloud. Without it the attestation step fails and the release stays
a draft; in that case either make the repo public, or remove the attestation
step and rely on `checksums.txt` plus reproducible builds.

## Installing a pinned version

Pick an exact tag and never use `latest`. With the GitHub CLI (also works for a
private repository, using your `gh auth login` session):

```bash
VERSION=vX.Y.Z
ASSET=linear-cli_Darwin_arm64.tar.gz   # or _Darwin_x86_64, _Linux_x86_64, _Linux_arm64, _Windows_x86_64.zip

gh release download "$VERSION" --repo Relynk/linear-cli \
  --pattern "$ASSET" --pattern checksums.txt

# 1. Checksum
sha256sum --ignore-missing -c checksums.txt        # macOS: shasum -a 256 --ignore-missing -c checksums.txt

# 2. Provenance: built by our release workflow, from this tag
gh attestation verify "$ASSET" --repo Relynk/linear-cli \
  --signer-workflow Relynk/linear-cli/.github/workflows/release.yml \
  --source-ref "refs/tags/$VERSION" \
  --deny-self-hosted-runners

tar -xzf "$ASSET" linear
gh attestation verify linear --repo Relynk/linear-cli   # the binary itself is attested too
install -m 0755 linear ~/.local/bin/linear              # or /usr/local/bin
linear --version
```

To pin a version for a team, record the tag **and** the SHA-256 of the
archive (from `checksums.txt`) in your setup script or dotfiles and check it
with `sha256sum -c` before installing; a hash pin still holds even if a
release were tampered with later.

Homebrew: `Formula/linear-cli.rb` points at the Relynk releases. If the
repository is private, Homebrew cannot download the assets without
authentication; use the `gh` steps above instead.

## Reproducing a release build

Anyone can rebuild a release binary from source and compare hashes:

```bash
git clone https://github.com/Relynk/linear-cli && cd linear-cli
git checkout vX.Y.Z          # clean checkout: local changes change the binary
make release-local GOOS=darwin GOARCH=arm64
# prints the sha256 of bin/release/linear; compare with the release binary:
shasum -a 256 bin/release/linear /path/to/extracted/linear
```

`make release-local` uses the exact Go version from `go.mod` (the `go`
command downloads and checksum-verifies it if needed) and the same flags as
`.goreleaser.yml` (`CGO_ENABLED=0`, `-trimpath`, `-s -w`, version ldflag).
The binary also embeds the commit it was built from:

```bash
go version -m linear | grep -E 'go1|vcs\.(revision|modified)'
```

`vcs.modified` must be `false` and `vcs.revision` must be the tag's commit.

## Updating pinned tools

Pins live in `.github/workflows/*.yml` (actions by commit SHA, GoReleaser and
govulncheck by exact version) and in `go.mod` / `go.sum`. When bumping:

- resolve the tag to a commit yourself
  (`git ls-remote https://github.com/actions/checkout refs/tags/vX.Y.Z`) and
  keep the `# vX.Y.Z` comment in sync;
- prefer releases that are at least a couple of weeks old;
- for the Go toolchain, bump the `go` (or add a `toolchain`) line in `go.mod`;
  CI and releases follow it automatically.
