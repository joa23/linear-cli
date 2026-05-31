# Version from git tag or commit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/joa23/linear-cli/internal/cli.Version=$(VERSION)"

# Install location. Defaults to /opt/homebrew on Apple Silicon (Homebrew default),
# otherwise /usr/local. Override with: make install PREFIX=/usr/local
PREFIX ?= $(shell if [ -d /opt/homebrew ]; then echo /opt/homebrew; else echo /usr/local; fi)
BINDIR ?= $(PREFIX)/bin

# Build CLI
build:
	go build $(LDFLAGS) -o bin/linear ./cmd/linear

# Install the locally-built binary to $(BINDIR)/linear
install: build
	@mkdir -p $(BINDIR)
	install -m 0755 bin/linear $(BINDIR)/linear
	@echo "Installed $$($(BINDIR)/linear --version 2>/dev/null || echo linear) to $(BINDIR)/linear"

# Remove the installed binary
uninstall:
	rm -f $(BINDIR)/linear
	@echo "Removed $(BINDIR)/linear"

# Run unit tests (excludes integration tests)
test:
	go test -v ./...

# Run integration tests (requires LINEAR_TOKEN)
integration-test:
	go test -v -tags=integration ./...

# Check test coverage
coverage:
	go test -cover ./...

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	@if ! command -v claude >/dev/null 2>&1; then \
		npm install -g @anthropic-ai/claude-cli; \
	else \
		echo "✓ Claude CLI already installed"; \
	fi
	@echo "✓ Dependencies installed"

# Setup (install dependencies)
setup: install-deps
	@echo "✓ Setup complete!"

# Clean build artifacts
clean:
	rm -f bin/linear

# Show help
help:
	@echo "Linear CLI"
	@echo ""
	@echo "Commands:"
	@echo "  make build            - Build CLI binary"
	@echo "  make install          - Build + install to $(BINDIR)/linear (override with PREFIX=...)"
	@echo "  make uninstall        - Remove $(BINDIR)/linear"
	@echo "  make test             - Run unit tests"
	@echo "  make integration-test - Run integration tests (needs LINEAR_TOKEN)"
	@echo "  make coverage         - Check test coverage"
	@echo "  make setup            - Install dependencies"
	@echo "  make clean            - Clean build artifacts"

.PHONY: build install uninstall test integration-test coverage install-deps setup clean help
