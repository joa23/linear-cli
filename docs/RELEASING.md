# Releasing and verifying linear-cli

Relynk distributes one static binary per platform from the GitHub Releases of
[`Relynk/linear-cli`](https://github.com/Relynk/linear-cli). Each release is:

- **built only in CI**, by `.github/workflows/release.yml`, from a tag
  (`vX.Y.Z-relynk.N`);
- **locked**: the Go toolchain is exactly the one in `go.mod` (`toolchain go1.27.4`; bump it for Go security releases)
  (`GOTOOLCHAIN=local`), dependencies come from `go.sum` and are checked with
  `go mod verify` (nothing is upgraded or tidied at release time), and every
  GitHub Action and GoReleaser itself is pinned to an exact version/commit;
- **static and reproducible**: `CGO_ENABLED=0`, `-trimpath`, commit-derived
  timestamps, so the same tag gives byte-identical binaries and archives;
- **verifiable**: `checksums.txt` (SHA-256) for every archive. Consumers pin
  the SHA-256 of the archive they use, and anyone can rebuild from the tag and
  compare hashes. (No GitHub artifact attestations: the repository is private
  and they would need GitHub Enterprise Cloud.)

## Cutting a release

1. Make sure `main` is green in CI (`.github/workflows/ci.yml`).
2. Update `CHANGELOG.md`, commit, and push.
3. Tag and push the tag. Relynk tags carry a `-relynk.N` suffix on top of the
   upstream version they are based on, so they never collide with upstream
   tags (`prerelease: false` in `.goreleaser.yml` keeps them normal releases):
   ```bash
   git tag -s v1.12.0-relynk.1 -m "v1.12.0-relynk.1"   # -s: signed tag, recommended
   git push origin v1.12.0-relynk.1
   ```
4. The workflow then
   1. runs `go mod verify`, `go mod tidy -diff`, `go vet` and `go test` with a
      read-only token;
   2. checks the Go toolchain matches `go.mod`, builds all platforms with
      GoReleaser and publishes the release with `checksums.txt`.
5. Update the pinned version and SHA-256 in the container build (below) from
   `gh release download <tag> --repo Relynk/linear-cli --pattern checksums.txt --output -`.

Never run `goreleaser release` locally; CI is the only source of release
artifacts.

Recommended repository settings (admin):
- a tag ruleset protecting `v*` (only maintainers can create, nobody can
  move or delete tags);
- the `release` environment (created on first run) with required reviewers
  and "protected tags only";
- branch protection on `main`;
- **immutable releases** enabled, so published assets and tags cannot be
  replaced.

## Installing a pinned version (container build)

Developers get the binary through the sandbox container build. Pin an exact
tag **and** the SHA-256 of the archive, and check the hash in the build. The
hash is taken from `checksums.txt` once, when the version is approved, and
committed with the build definition; a later change to the release assets
then fails the build instead of shipping a different binary.

The repository is private, so the download needs a GitHub token with read
access to `Relynk/linear-cli` (fine-grained PAT with "Contents: read", or a
CI token). Pass it as a build secret so it never ends up in an image layer:

```dockerfile
# syntax=docker/dockerfile:1
ARG LINEAR_CLI_VERSION=v1.12.0-relynk.1
# SHA-256 of the archive for each architecture, from that release's checksums.txt
ARG LINEAR_CLI_SHA256_AMD64=<sha256 of linear-cli_Linux_x86_64.tar.gz>
ARG LINEAR_CLI_SHA256_ARM64=<sha256 of linear-cli_Linux_arm64.tar.gz>
ARG TARGETARCH

RUN --mount=type=secret,id=gh_token,required=true \
    set -eu; \
    case "$TARGETARCH" in \
      amd64) asset=linear-cli_Linux_x86_64.tar.gz; sha="$LINEAR_CLI_SHA256_AMD64" ;; \
      arm64) asset=linear-cli_Linux_arm64.tar.gz;  sha="$LINEAR_CLI_SHA256_ARM64" ;; \
      *) echo "unsupported arch $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    tok="$(cat /run/secrets/gh_token)"; \
    api=https://api.github.com/repos/Relynk/linear-cli; \
    id="$(curl -fsSL -H "Authorization: Bearer $tok" "$api/releases/tags/$LINEAR_CLI_VERSION" \
          | jq -r --arg n "$asset" '.assets[] | select(.name==$n) | .id')"; \
    curl -fsSL -H "Authorization: Bearer $tok" -H "Accept: application/octet-stream" \
         -o "/tmp/$asset" "$api/releases/assets/$id"; \
    echo "$sha  /tmp/$asset" | sha256sum -c -; \
    tar -xzf "/tmp/$asset" -C /usr/local/bin linear; \
    chmod 0755 /usr/local/bin/linear; rm -f "/tmp/$asset"; \
    linear --version
```

```bash
docker build --secret id=gh_token,env=GH_TOKEN .
```

If the build image has the GitHub CLI, the two `curl` calls can be replaced
with `GH_TOKEN="$tok" gh release download "$LINEAR_CLI_VERSION" --repo
Relynk/linear-cli --pattern "$asset" --dir /tmp`. Keep the `sha256sum -c`
line either way: it is the actual pin.

Outside a container (e.g. a one-off on a laptop):

```bash
VERSION=v1.12.0-relynk.1
ASSET=linear-cli_Darwin_arm64.tar.gz   # or _Darwin_x86_64, _Linux_x86_64, _Linux_arm64, _Windows_x86_64.zip
gh release download "$VERSION" --repo Relynk/linear-cli --pattern "$ASSET" --pattern checksums.txt
sha256sum --ignore-missing -c checksums.txt   # macOS: shasum -a 256 --ignore-missing -c checksums.txt
tar -xzf "$ASSET" linear && install -m 0755 linear ~/.local/bin/linear
```

## Reproducing a release build

Anyone can rebuild a release binary from source and compare hashes:

```bash
git clone https://github.com/Relynk/linear-cli && cd linear-cli
git checkout v1.12.0-relynk.1   # clean checkout: local changes change the binary
make release-local GOOS=linux GOARCH=amd64
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
