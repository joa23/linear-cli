# Version from git tag or commit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
MODULE  := github.com/joa23/linear-cli

# Toolchain pinned in go.mod (toolchain line, else go line); release builds use exactly it.
GO_VERSION := $(shell awk '/^toolchain go/{sub(/^go/,"",$$2); t=$$2} /^go /{g=$$2} END{print (t!=""?t:g)}' go.mod)

# Static, path-independent builds. These flags must match .goreleaser.yml so
# that `make release-local` reproduces the published release binaries.
export CGO_ENABLED := 0
export GOFLAGS     := -mod=readonly
BUILD_FLAGS        := -trimpath
LDFLAGS            := -ldflags "-X $(MODULE)/internal/cli.Version=$(VERSION)"

# Build CLI (local toolchain)
build:
	go build $(BUILD_FLAGS) $(LDFLAGS) -o bin/linear ./cmd/linear

# Run unit tests (excludes integration tests)
test:
	go test -v ./...

# Run integration tests (requires LINEAR_TOKEN)
integration-test:
	go test -v -tags=integration ./...

# Check test coverage
coverage:
	go test -cover ./...

# Verify dependencies against go.sum and run static checks
verify:
	go mod verify
	go vet ./...

# ---------------------------------------------------------------------------
# Reproducible release build
#
# Rebuilds the binary exactly as the release workflow does, so you can check
# that a published binary matches the source at a tag:
#
#   git checkout vX.Y.Z           # clean tree, no local changes
#   make release-local GOOS=darwin GOARCH=arm64
#   # compare with the `linear` binary from the release archive:
#   shasum -a 256 bin/release/linear ./linear
#
# RELEASE_VERSION must equal the tag without the leading "v" (GoReleaser's
# {{.Version}}). The Go toolchain is the exact one from go.mod (fetched and
# checksum-verified by the go command if it is not installed).
# ---------------------------------------------------------------------------
RELEASE_VERSION ?= $(patsubst v%,%,$(shell git describe --tags --exact-match 2>/dev/null))
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXE    := $(if $(filter windows,$(GOOS)),.exe,)

release-local:
	@if [ -z "$(RELEASE_VERSION)" ]; then \
		echo "HEAD is not a tag: check out vX.Y.Z or pass RELEASE_VERSION=X.Y.Z"; exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "warning: working tree has local changes; the binary will not match the release"; \
	fi
	GOTOOLCHAIN=go$(GO_VERSION) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(BUILD_FLAGS) \
		-ldflags "-s -w -X $(MODULE)/internal/cli.Version=$(RELEASE_VERSION)" \
		-o bin/release/linear$(EXE) ./cmd/linear
	@cd bin/release && (sha256sum linear$(EXE) 2>/dev/null || shasum -a 256 linear$(EXE))

# Clean build artifacts
clean:
	rm -rf bin/ dist/

# Show help
help:
	@echo "Linear CLI"
	@echo ""
	@echo "Commands:"
	@echo "  make build            - Build CLI binary (static, -trimpath)"
	@echo "  make test             - Run unit tests"
	@echo "  make integration-test - Run integration tests (needs LINEAR_TOKEN)"
	@echo "  make coverage         - Check test coverage"
	@echo "  make verify           - go mod verify + go vet"
	@echo "  make release-local    - Reproduce a release binary (see docs/RELEASING.md)"
	@echo "  make clean            - Clean build artifacts"

.PHONY: build test integration-test coverage verify release-local clean help
