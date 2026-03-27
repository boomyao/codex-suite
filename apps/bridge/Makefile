GO ?= go
NODE ?= node
VERSION ?= dev
PLATFORM ?= $(shell $(GO) env GOOS)-$(shell arch | sed 's/aarch64/arm64/; s/x86_64/x64/; s/amd64/x64/')
OUT_DIR ?= dist

.PHONY: build run check print-config init-config package-release

build:
	$(GO) build ./...

run:
	$(GO) run ./cmd/codex-bridge start

check:
	$(GO) fmt ./...
	$(GO) build ./...

print-config:
	$(GO) run ./cmd/codex-bridge print-config

init-config:
	$(GO) run ./cmd/codex-bridge init-config

package-release:
	$(NODE) ./scripts/package_release.mjs \
		--version "$(VERSION)" \
		--platform "$(PLATFORM)" \
		--output-dir "$(OUT_DIR)" \
		--pointer-manifest "$(POINTER_MANIFEST)"
