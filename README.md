# Codex Suite

`codex-suite` is a monorepo that contains the mobile client and the bridge service used to expose Codex sessions to mobile devices.

## Layout

- `apps/mobile`
  - React Native / Expo mobile client
- `apps/bridge`
  - Go bridge service that serves shell assets and proxies Codex protocol traffic

## Working Model

The mobile app currently expects a single external service address that:

- serves shell UI assets under `/ui/index.html`
- exposes websocket transport for Codex protocol traffic
- can optionally manage or proxy a `codex app-server`

In this suite, that service lives in `apps/bridge`.

## Quick Start

Run the bridge:

```bash
cd apps/bridge
go run ./cmd/codex-bridge start
```

Run the mobile app:

```bash
cd apps/mobile
npm install
npm run ios
```

For Android:

```bash
cd apps/mobile
npm run android
```

## Notes

- `apps/mobile/ios` and `apps/mobile/android` are not committed here; Expo will regenerate them when needed.
- The imported `apps/bridge` content reflects the current working tree from `/Users/boomyao/lab/codex-bridge`, including files that had not been committed there yet.
