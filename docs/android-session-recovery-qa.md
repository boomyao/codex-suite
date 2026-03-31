# Android Session Recovery QA

This note covers the expected Android UX for saved Codex Mobile sessions across backgrounding, reconnect attempts, and manual exit.

## Product intent

- If the user was actively inside a workspace and the app is backgrounded, returning to the app should feel continuous.
- Cold launch should land on the native session page instead of immediately reopening the last session.
- If a session drops while the user is already in that session flow, the native session UI may attempt a single automatic reconnect.
- If the user explicitly backs out to the native session picker, that is treated as an intentional exit from the workspace. A later launch should not auto-reconnect until the user chooses a saved session again.

## Expected UX

### 1. Cold launch with an active saved session

- App opens on the native session page instead of jumping into a workspace.
- Saved sessions are shown as reconnectable cards.
- The user decides when to reopen a saved session.

### 2. Session error while the app is active

- The native error/session UI appears.
- The status copy changes to `Trying to restore this session`.
- The app attempts one automatic reconnect for the active session.
- If that reconnect fails, the session page stays visible with manual retry controls.

### 3. App sent to background and reopened while still connected

- The current task is brought back to front.
- The existing workspace remains visible.
- The user should not see the native session page flash during normal resume.

### 4. User manually exits workspace with back

- The native home screen reappears with saved sessions.
- The user sees trusted sessions and `Scan QR`.
- Auto-resume is disabled for the active session until the user explicitly opens a session again.

### 5. Cold launch after manual exit

- App stays on the native home screen.
- Saved sessions are shown as reconnectable cards.
- The app does not immediately reopen the last workspace.

## Manual verification steps

1. Connect Codex Mobile to a trusted desktop session.
2. Kill the app process and relaunch it.
3. Confirm the app opens on the native session page and does not auto-connect.
4. Open a saved session manually.
5. Press Home, reopen the app from recents or launcher, and confirm the workspace is still present.
6. Trigger or simulate a session load failure and confirm the session page shows `Trying to restore this session`, followed by one automatic reconnect attempt.
7. Press Back from the workspace to return to the native home screen.
8. Kill the app process and relaunch it again.
9. Confirm the app stays on the native home screen and does not auto-reconnect.

## adb-driven validation

The following commands were used during validation:

```bash
./gradlew app:installDebug
adb shell am start -n com.boomyao.codexmobile/.nativehost.NativeHostActivity
adb exec-out uiautomator dump /dev/tty > /tmp/codex-mobile.xml
adb exec-out screencap -p > /tmp/codex-mobile.png
adb shell input keyevent KEYCODE_HOME
adb shell input keyevent KEYCODE_BACK
adb shell am force-stop com.boomyao.codexmobile
```

## Verified result

- Cold launch now stays on the native session page instead of auto-opening the latest session.
- Background and reopen while connected still returns to the live WebView.
- Manual exit returns to the native home screen and keeps later cold launches on that page.
- Automatic reconnect is now scoped to session-error handling rather than app launch.
