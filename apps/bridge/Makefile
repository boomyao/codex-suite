GO ?= go

.PHONY: build run check print-config init-config

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
