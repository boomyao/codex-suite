# codex-bridge

This directory contains the bridge service used by Codex Suite.

`codex-bridge` is a headless Go service that manages:

- a local websocket bridge
- a managed or remote `codex app-server`

It does not embed or launch `Codex.app`. It can serve a bundled `desktop-webview` directory so Codex Mobile works without installing `Codex.app`.

## Scope

The Go implementation covers the main headless runtime path:

- CLI entrypoint
- config file loading and validation
- environment-variable overrides
- managed local `codex app-server`
- remote upstream mode
- websocket bridge
- desktop UI asset hosting under `/ui/*`
- `healthz`, `readyz`, and `status`
- direct RPC on `/codex-mobile/rpc`

## Local Runtime Store

On startup, `codex-bridge` now prefers a self-managed local runtime under the config directory:

```text
~/.codex-bridge/
  config.json
  current.json
  runtimes/
    darwin-arm64/
      local-20260326T130237Z/
        desktop-webview/
        bin/
          codex
        manifest.json
```

`current.json` points to the active runtime, and each `manifest.json` records the imported sources and hashes.

When no local runtime is active, `codex-bridge` will try to import:

- `desktop-webview` from an explicit `--desktop-webview-root`, env override, or bundled adjacent directory
- `app-server` from `--app-server-bin`, `PATH`, or macOS `Codex.app/Contents/Resources/codex`

After import, bridge and managed app-server run from the copied local runtime rather than from the original source path.

## Bundle Layout

For end-user distribution, place the desktop web bundle next to the `codex-bridge` binary:

```text
codex-bridge
desktop-webview/
  index.html
  assets/...
```

If `desktopWebviewRoot` is not configured, `codex-bridge` will look for:

- `<executable dir>/desktop-webview`
- `<executable dir>/resources/desktop-webview`
- macOS app-style `<executable dir>/../Resources/desktop-webview`
- development fallbacks under the current working directory

You can still override this explicitly with `--desktop-webview-root` or `CODEX_BRIDGE_DESKTOP_WEBVIEW_ROOT`.

For a fully self-contained macOS layout, this also works:

```text
codex-bridge
desktop-webview/
codex
```

The first launch will copy these into the local runtime store and then use the copied versions.

## Development

Build:

```bash
go build ./...
```

Run:

```bash
go run ./cmd/codex-bridge start
```

Print the effective config:

```bash
go run ./cmd/codex-bridge print-config
```

Initialize a config file:

```bash
go run ./cmd/codex-bridge init-config
```

Or use the convenience targets:

```bash
make check
make run
make print-config
make init-config
```

## CLI

```bash
codex-bridge [start] [options]
codex-bridge print-config [options]
codex-bridge init-config [options]
```

Important options:

- `--config /absolute/path/to/config.json`
- `--managed`
- `--remote ws://127.0.0.1:9876`
- `--bridge-host 127.0.0.1`
- `--bridge-port 8787`
- `--desktop-webview-root /absolute/path/to/desktop-webview`
- `--app-server-port 9876`
- `--app-server-bin /absolute/path/to/codex`
- `--app-server-arg <arg>`
- `--auto-restart`
- `--no-auto-restart`
- `--restart-delay-ms 1500`

## Config File

Default config path:

```text
~/.codex-bridge/config.json
```

Example:

```json
{
  "runtimeMode": "managed",
  "remoteUpstreamUrl": "",
  "bridgeHost": "127.0.0.1",
  "bridgePort": 8787,
  "desktopWebviewRoot": "",
  "uiPathPrefix": "/ui",
  "appServerPort": 9876,
  "appServerBin": "codex",
  "appServerArgs": [],
  "autoRestart": true,
  "restartDelayMs": 1500
}
```

Remote mode example:

```json
{
  "runtimeMode": "remote",
  "remoteUpstreamUrl": "ws://10.0.0.5:9876",
  "bridgeHost": "127.0.0.1",
  "bridgePort": 8787,
  "desktopWebviewRoot": "",
  "uiPathPrefix": "/ui",
  "appServerPort": 9876,
  "appServerBin": "codex",
  "appServerArgs": [],
  "autoRestart": true,
  "restartDelayMs": 1500
}
```

## Environment Variables

Environment variables can override config values:

- `CODEX_BRIDGE_HOST`
- `CODEX_BRIDGE_PORT`
- `CODEX_BRIDGE_DESKTOP_WEBVIEW_ROOT`
- `CODEX_BRIDGE_UI_PATH_PREFIX`
- `CODEX_BRIDGE_UPSTREAM_URL`
- `CODEX_MANAGE_APP_SERVER`
- `CODEX_APP_SERVER_PORT`
- `CODEX_APP_SERVER_BIN`
- `CODEX_APP_SERVER_EXTRA_ARGS`
- `CODEX_AUTO_RESTART_RUNTIME`
- `CODEX_RUNTIME_RESTART_DELAY_MS`

## Endpoints

When running locally with defaults:

- websocket bridge: `ws://127.0.0.1:8787`
- desktop UI: `http://127.0.0.1:8787/ui/index.html`
- health: `http://127.0.0.1:8787/healthz`
- ready: `http://127.0.0.1:8787/readyz`
- status: `http://127.0.0.1:8787/status`
- direct RPC: `http://127.0.0.1:8787/codex-mobile/rpc`
