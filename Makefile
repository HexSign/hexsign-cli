VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PKG     := github.com/hexsign/hexsign-cli

# HEXSIGN_CLI_CLIENT_ID is injected at build time so distributed binaries are
# zero-config. Pass it via env or `make build HEXSIGN_CLI_CLIENT_ID=…`. Local
# dev builds without it still work — the CLI falls back to the env var at
# runtime, with a clear error if neither is set.
HEXSIGN_CLI_CLIENT_ID ?=

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X $(PKG)/internal/config.DefaultUserClientID=$(HEXSIGN_CLI_CLIENT_ID)

.PHONY: build install test tidy clean release-build

build:
	@mkdir -p bin
	go build -trimpath -ldflags='$(LDFLAGS)' -o bin/hexsign .

install:
	go install -trimpath -ldflags='$(LDFLAGS)' .

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin dist

# Cross-compile release binaries. HEXSIGN_CLI_CLIENT_ID must be set.
release-build:
	@if [ -z "$(HEXSIGN_CLI_CLIENT_ID)" ]; then \
		echo "error: HEXSIGN_CLI_CLIENT_ID must be set for release builds (terraform output cli_user_client_id)"; \
		exit 1; \
	fi
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags='$(LDFLAGS)' -o dist/hexsign-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags='$(LDFLAGS)' -o dist/hexsign-darwin-amd64 .
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags='$(LDFLAGS)' -o dist/hexsign-linux-arm64 .
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags='$(LDFLAGS)' -o dist/hexsign-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='$(LDFLAGS)' -o dist/hexsign-windows-amd64.exe .
