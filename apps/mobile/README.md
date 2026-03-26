# Codex Mobile

This directory contains the React Native / Expo mobile client for Codex Suite.

The remote service that serves the shell UI and proxies Codex protocol traffic
lives in `../bridge`.

## What is implemented

- Codex-style mobile shell with:
  - thread drawer
  - dense timeline cards
  - inspector drawer
  - connection/settings sheet
- WebView-hosted shell with a mobile `electronBridge` shim.
- Live websocket integration for:
  - `thread/list`
  - `thread/read`
  - `thread/resume`
  - `thread/start`
  - `turn/start`
  - `turn/interrupt`
  - `account/read`
  - `account/rateLimits/read`
- Streaming notification support for:
  - thread status
  - turn start/completion
  - turn diff updates
  - turn plan updates
  - agent message deltas
  - reasoning deltas
  - command/file output deltas
  - token usage updates
  - account and rate-limit updates
- Mobile-side handling for server requests:
  - command approval
  - file-change approval
  - permission approval
  - tool user input
  - MCP elicitation
- Local sample thread state for offline shell rendering before a live backend is connected.

## Repository scope

- `App.tsx`
  - owns native state, persistence, alerts, remote shell loading, and WebView message handling
- `src/appServer.ts`
  - contains the default remote service endpoint
- `src/sampleData.ts`
  - provides local fake threads/messages

Not included in this app directory:

- remote service / gateway implementation
- desktop Electron shell
- hosted shell UI asset bundle

## Run it

```bash
npm install
npm run ios
```

For Android:

```bash
npm run android
```

For Metro only:

```bash
npm start
```

## How to use the prototype

- Start a Codex app server:

```bash
codex app-server --listen ws://127.0.0.1:9876
```

- Point the app at a compatible external service endpoint that serves `/ui/index.html` and forwards Codex protocol websocket traffic.
- The app defaults to `ws://127.0.0.1:8787` because the current deployment shape still expects one service address for both shell assets and transport.
- Open `Threads` to switch sessions.
- Open `Inspect` to view account, rate-limit, diff, token, request, and transport state.
- Open `Settings` to set `Server Endpoint`: the address that serves `/ui/index.html` plus the websocket endpoint used by the mobile shell.
- Use the composer to continue turns or start a new thread.
- Handle approvals and input requests from the Inspector panel when Codex pauses for client input.

If you are connecting from a physical phone instead of a simulator, replace `127.0.0.1` with the host machine LAN IP.

If native folders are missing, regenerate them through Expo. In this suite,
`ios/` and `android/` are intentionally left out of version control.

## Why this is remote-first

The desktop Codex app depends on Electron main-process features, PTY execution, filesystem access, window management, and native modules. This mobile client currently stays close to Codex desktop by loading shell assets from a remote service and steering sessions over `app-server`, instead of trying to host the full local execution stack on-device.
