# Codex Mobile Native Host

This note replaces the previous assumption that `codex-mobile` should remain a React Native app.

## Decision

`codex-mobile` should become:

- Android native host
- iOS native host
- `WebView` container for the existing bridge UI
- direct bridge client

The web content remains the product UI. The mobile app becomes a systems shell.

## Why

- the hard part on mobile is host integration, not reimplementing Tailscale
- the app should connect to bridge endpoints directly over HTTP and WebSocket
- if bridge is exposed on tailnet, the system Tailscale app should own reachability
- QR enrollment, app lifecycle, background behavior, and permission handling are all platform-native concerns
- keeping React Native as a thin wrapper would still add another runtime without reducing the hard parts

## Current repository direction

### Android

`apps/mobile/android` now has a native launcher activity:

- `.nativehost.NativeHostActivity`
- `BridgeProfileStore`
- bridge enrollment parsing
- bridge pairing and `WebView` loading
- direct bridge bootstrap and websocket transport

### iOS

`apps/mobile/ios-native` now contains the initial native host source set:

- `CodexMobileHostApp.swift`
- `ContentView.swift`
- `CodexWebView.swift`
- `BridgeProfileStore.swift`
- `EnrollmentPayload.swift`
- `BridgeAPI.swift`

`apps/mobile/ios-native` now also includes a buildable Xcode project:

- `CodexMobileHost.xcodeproj`
- shared `CodexMobileHost` scheme

The iOS app now follows the same direct bridge model as Android and does not depend on an embedded tailnet runtime.

## Electron WebView compatibility notes

Running the existing `codex.app` web UI inside a mobile `WebView` is not just a packaging task.
The desktop web app assumes an Electron host, a desktop-style app-server session, and a set of local host APIs.

The current Android implementation works because it adds a compatibility shell around those assumptions.
iOS should reuse the same categories of behavior instead of rediscovering them one by one.

### 1. Host bridge

The web app expects a host bridge for messages between renderer and native shell.

What exists now:

- bridge preload injects a mobile host shim in `apps/bridge/internal/bridge/ui.go`
- Android exposes the native bridge in `apps/mobile/android/app/src/main/java/com/boomyao/codexmobile/nativehost/NativeHostActivity.kt`

Rule:

- keep a single mobile host bridge contract
- do not reintroduce `ReactNativeWebView` compatibility paths

### 2. App-server transport split

Not every web app call has the same transport shape.

What exists now:

- long-lived interactive requests use a persistent websocket session in `apps/mobile/android/app/src/main/java/com/boomyao/codexmobile/nativehost/AppServerWebSocketClient.kt`
- direct one-shot host calls still use `/codex-mobile/rpc` in `apps/bridge/internal/bridge/bridge.go`

Why:

- `turn/start` and incremental assistant output need ordered notifications on the same session
- startup configuration and local host lookups are simpler as direct request/response calls

Rule:

- keep websocket as the primary transport for interactive Codex turns
- keep direct RPC only for startup and host-local capabilities

### 3. Host-local state hydration

The web UI is not actually stateless on startup.
It expects persisted atoms, workspace roots, pinned threads, and related shell state to exist before the first screen fully settles.

What exists now:

- bridge exposes `get-global-state-snapshot` in `apps/bridge/internal/bridge/bridge.go`
- Android bootstraps from that snapshot in `apps/mobile/android/app/src/main/java/com/boomyao/codexmobile/nativehost/NativeHostActivity.kt`

Why:

- without bootstrap hydration, mobile looks like a fresh install even when desktop already has projects and threads

Rule:

- always hydrate shell state before loading the main `WebView`
- treat bridge global state as the source of truth

### 4. Workspace and thread reconciliation

The mobile host still needs a small amount of local reconciliation even after bootstrap.

What exists now:

- Android merges workspace roots from `thread/list`, `thread/read`, `thread/start`, and `thread/resume`
- Android keeps a fallback-only turn completion reconciliation path in `NativeHostActivity.kt`

Why:

- the desktop web app derives parts of shell state from session results, not just from a single persisted blob
- websocket drops or timing gaps can otherwise leave UI stuck in `Thinking`

Rule:

- keep turn reconciliation as a fallback, not the main completion path
- continue merging workspace roots from upstream results until the web app stops depending on that behavior

### 5. Codex protocol and fetch interception

The desktop web app still issues host-oriented fetches such as `vscode://codex/...`.

What exists now:

- bridge preload intercepts protocol fetches in `apps/bridge/internal/bridge/ui.go`
- Android maps those requests to local methods or bridge direct RPC in `apps/mobile/android/app/src/main/java/com/boomyao/codexmobile/nativehost/NativeHostActivity.kt`

Why:

- the web app was not originally written against a pure browser transport model

Rule:

- keep a thin protocol translation layer
- do not duplicate business logic in both preload and native host
- prefer moving method implementations back to bridge over adding new native stubs

### 6. Tailnet ownership

Tailnet may still be part of the deployment, but it should not be implemented inside the mobile app.

What exists now:

- mobile stores plain bridge profiles and connects directly to the advertised bridge endpoint
- legacy `codex-mobile-enrollment` payloads are downgraded into normal bridge profiles for compatibility

Why:

- embedded tailnet added a large amount of platform-specific lifecycle, secret storage, and proxy complexity
- system Tailscale already owns device auth, network extension/VPN integration, and tailnet reachability

Rule:

- keep the mobile host unaware of tailnet implementation details
- treat `.ts.net` and Tailscale IP endpoints as ordinary bridge addresses
- if a user needs tailnet reachability, rely on the installed Tailscale app

### 7. Startup race handling

Bridge startup and mobile startup are not perfectly synchronized.

What exists now:

- Android retries bootstrap snapshot fetches in `apps/mobile/android/app/src/main/java/com/boomyao/codexmobile/nativehost/BridgeApi.kt`

Why:

- without retry, bridge restarts can produce partial startup failures and empty state hydration

Rule:

- retry bridge bootstrap reads
- assume mobile may connect while bridge is still warming up

## Compatibility debt to avoid growing

These items are still present, but should shrink over time:

- host-side direct RPC stubs that duplicate bridge behavior
- protocol translation around `vscode://codex/...`
- third-party environment stubs in bridge preload
- mobile-specific visual patches mixed into the same preload script as host protocol logic

If iOS needs a new workaround, prefer one of these outcomes:

- move the implementation into bridge so Android and iOS share it
- define a host-level contract that both platforms implement the same way

Avoid:

- adding iOS-only copies of Android business stubs
- restoring React Native compatibility layers
- pushing more Codex session logic into the mobile shell

## iOS bring-up order

When iOS is wired up, the safest order is:

1. Recreate the same host bridge contract as Android.
2. Reuse bridge bootstrap hydration before first `WebView` load.
3. Reuse websocket-backed interactive transport for Codex turns.
4. Add only the minimum host-local RPC surface needed for startup and shell state.
5. Keep the mobile host on direct bridge transport and avoid reintroducing an embedded network stack.
6. Only then address visual or gesture-specific mobile patches.

## Next steps

1. Move bridge onboarding fully into native Android UI, including camera QR scan.
2. Keep shrinking tailnet-specific code that is no longer on the native host path.
3. Improve user-facing guidance when the system Tailscale app is required but not connected.
4. Keep refining the native enrollment and direct bridge flow.
