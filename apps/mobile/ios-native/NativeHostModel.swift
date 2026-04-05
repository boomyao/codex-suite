import Foundation
import ImageIO
import PhotosUI
import SwiftUI
import UIKit
import UniformTypeIdentifiers

@MainActor
final class NativeHostModel: ObservableObject {
    private static let fullDesktopFileAccessConfigurationKey = "mobile.fullDesktopFileAccess"

    @Published private(set) var profiles: [BridgeProfile] = []
    @Published var activeProfile: BridgeProfile?
    @Published private(set) var chromeState: ShellChromeState = .disconnected
    @Published private(set) var statusMessage = "Choose how to connect"
    @Published private(set) var currentConnectionStage: NativeHostConnectionStage?
    @Published private(set) var currentConnectionRequiresPairing = false
    @Published var themeMode: ThemeMode
    @Published var fullDesktopFileAccessEnabled: Bool
    @Published var browserSessionSheetPresented = false
    @Published var desktopSessionSheetPresented = false
    @Published var sessionsSheetPresented = false
    @Published var settingsSheetPresented = false
    @Published var scannerPresented = false
    @Published var systemColorScheme: ColorScheme = .dark
    @Published var profilePendingReset: BridgeProfile?

    let webBridge = NativeHostWebBridge()

    private let profileStore: BridgeProfileStore
    private let preferencesStore: NativeHostPreferencesStore
    private let runtime: NativeHostBridgeRuntime
    private let desktopThumbnailCache = NativeHostDesktopThumbnailCache()

    private var activeLoadTarget: BridgeLoadTarget?
    private var bridgeLoadGeneration = 0
    private var autoReconnectProfileID: String?
    private var autoReconnectAttemptCount = 0
    private var autoReconnectTask: Task<Void, Never>?

    init(
        profileStore: BridgeProfileStore = BridgeProfileStore(),
        preferencesStore: NativeHostPreferencesStore = NativeHostPreferencesStore(),
        runtime: NativeHostBridgeRuntime? = nil
    ) {
        self.profileStore = profileStore
        self.preferencesStore = preferencesStore
        self.runtime = runtime ?? NativeHostBridgeRuntime()
        self.themeMode = preferencesStore.readThemeMode()
        self.fullDesktopFileAccessEnabled = preferencesStore.isFullDesktopFileAccessEnabled()

        self.runtime.delegate = self
        self.runtime.setJavaScriptEvaluator { [weak webBridge] script in
            webBridge?.evaluateJavaScript(script)
        }
        webBridge.delegate = self
        NativeHostDebugLog.reset()
        NativeHostDebugLog.write("model-init")

        refreshProfiles()
        renderDisconnectedHome(message: "Choose how to connect")
    }

    var preferredColorScheme: ColorScheme? {
        themeMode.preferredColorScheme
    }

    var resolvedThemeName: String {
        switch themeMode {
        case .system:
            return systemColorScheme == .light ? "light" : "dark"
        case .light:
            return "light"
        case .dark:
            return "dark"
        }
    }

    var isConnected: Bool { chromeState == .connected }
    var isLoading: Bool { chromeState == .loading }
    var isError: Bool { chromeState == .error }
    var isAutoReconnectPending: Bool { chromeState == .error && autoReconnectTask != nil }
    var hasSavedProfiles: Bool { !profiles.isEmpty }
    var savedProfilesPreview: [BridgeProfile] { Array(profiles.prefix(2)) }
    var activeSessionForSheet: BridgeProfile? {
        if let activeProfile, !isTransientProfile(activeProfile) {
            return activeProfile
        }
        return profileStore.readActive()
    }
    var shouldShowWorkspace: Bool { activeProfile != nil && chromeState == .connected }
    var shouldKeepWebViewMounted: Bool {
        activeProfile != nil && (chromeState == .connected || chromeState == .loading || isAutoReconnectPending)
    }

    var loadingSummary: String {
        if let activeProfile {
            return "Preparing \(displayProfileLabel(activeProfile)) on this iPad."
        }
        return "Preparing your Codex session."
    }

    var connectionProgressValue: Double {
        let steps = connectionStages
        guard let currentConnectionStage,
              let index = steps.firstIndex(of: currentConnectionStage) else {
            return 0
        }
        return Double(index + 1) / Double(max(steps.count, 1))
    }

    var connectionStageLabels: [String] {
        connectionStages.map(\.title)
    }

    private var connectionStages: [NativeHostConnectionStage] {
        var steps: [NativeHostConnectionStage] = [.payloadReceived]
        if currentConnectionRequiresPairing {
            steps.append(.pairingDevice)
        }
        steps.append(.openingWorkspace)
        return steps
    }

    func handleColorSchemeChanged(_ colorScheme: ColorScheme) {
        systemColorScheme = colorScheme
        updateWebTheme()
    }

    func openScanner() {
        scannerPresented = true
    }

    func saveThemeMode(_ themeMode: ThemeMode) {
        self.themeMode = themeMode
        preferencesStore.writeThemeMode(themeMode)
        updateWebTheme()
    }

    func saveFullDesktopFileAccessEnabled(_ enabled: Bool) {
        fullDesktopFileAccessEnabled = enabled
        preferencesStore.setFullDesktopFileAccessEnabled(enabled)

        guard let activeProfile, let activeLoadTarget else {
            return
        }

        Task {
            do {
                try await synchronizeMobileHostSettings(
                    profile: activeProfile,
                    loadTarget: activeLoadTarget
                )
            } catch {
                NativeHostDebugLog.write("full-desktop-access-sync-error message=\(normalizeErrorMessage(error))")
            }
        }
    }

    func handleScannedPayload(_ rawPayload: String) {
        NSLog("CodexMobile scanned QR payload length=%d", rawPayload.count)
        NativeHostDebugLog.write("scanned-payload length=\(rawPayload.count)")
        scannerPresented = false
        importEnrollment(rawPayload)
    }

    func activateProfile(_ profile: BridgeProfile) {
        resetAutoReconnectState()
        profileStore.setActive(profile.id)
        refreshProfiles()
        openBridge(profile)
    }

    func reloadActiveBridge() {
        resetAutoReconnectState()
        guard let activeProfile else {
            renderDisconnectedHome(message: "Choose how to connect")
            return
        }
        openBridge(activeProfile)
    }

    func isTransientProfile(_ profile: BridgeProfile) -> Bool {
        profile.id.hasPrefix("pending")
    }

    func resetEnrollment(_ profile: BridgeProfile) {
        profilePendingReset = nil
        let nextProfile = profileStore.remove(profile.id)
        refreshProfiles()
        resetAutoReconnectState()

        if nextProfile == nil {
            activeProfile = nil
            activeLoadTarget = nil
            webBridge.loadBlank()
            renderDisconnectedHome(message: "Choose how to connect")
            return
        }

        activeProfile = nil
        activeLoadTarget = nil
        openBridge(nextProfile!)
    }

    func describeConnectingStatus(_ profile: BridgeProfile) -> String {
        let normalizedStatus = statusMessage.trimmingCharacters(in: .whitespacesAndNewlines)
        if !normalizedStatus.isEmpty {
            return normalizedStatus
        }
        return currentConnectionStage?.title ?? displayProfileDetail(profile)
    }

    func summarizeWorkspaceError(_ message: String) -> String {
        let normalized = message.trimmingCharacters(in: .whitespacesAndNewlines)
        if isAutoReconnectPending {
            return "This device lost contact with the desktop bridge. Codex Mobile will keep retrying until the bridge responds or you switch sessions."
        }
        if normalized.localizedCaseInsensitiveContains("invalid key") ||
            normalized.localizedCaseInsensitiveContains("fresh enrollment qr") ||
            normalized.localizedCaseInsensitiveContains("re-enrolled") {
            return "This saved setup is no longer valid. Generate a fresh QR code on desktop and try again."
        }
        if normalized.localizedCaseInsensitiveContains("expired") {
            return "This setup code expired before the workspace opened."
        }
        if normalized.isEmpty {
            return "This session stopped before the workspace opened. Retry, pick another saved session, or scan a fresh desktop QR code."
        }
        return compactLabel(normalized, maxLength: 96)
    }

    func workspaceErrorTitle(_ message: String) -> String {
        let normalized = message.trimmingCharacters(in: .whitespacesAndNewlines)
        if isAutoReconnectPending {
            return "Reconnecting to Desktop"
        }
        if normalized.localizedCaseInsensitiveContains("expired") {
            return "QR Code Expired"
        }
        return "Couldn’t open workspace"
    }

    func presentSessionsSheet() {
        sessionsSheetPresented = true
    }

    func presentBrowserSessionSheet() {
        browserSessionSheetPresented = true
    }

    func presentDesktopSessionSheet() {
        desktopSessionSheetPresented = true
    }

    func presentSettingsSheet() {
        settingsSheetPresented = true
    }

    func importEnrollment(_ rawJSON: String) {
        resetAutoReconnectState()
        let trimmed = rawJSON.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            statusMessage = "Scan a desktop QR code first."
            return
        }

        scannerPresented = false

        do {
            switch try EnrollmentParser.parse(rawJSON: trimmed) {
            case let .bridge(bridgeID, name, serverEndpoint, pairingCode, libp2pPeerID):
                NSLog("CodexMobile importEnrollment type=bridge endpoint=%@", serverEndpoint)
                NativeHostDebugLog.write("import type=bridge bridgeID=\(bridgeID ?? "<nil>") endpoint=\(serverEndpoint) pairing=\(pairingCode != nil) libp2p=\(libp2pPeerID ?? "<nil>")")
                updateConnectionProgress(
                    bridgeID: bridgeID,
                    profileName: name,
                    endpoint: serverEndpoint,
                    stage: .payloadReceived,
                    requiresPairing: pairingCode != nil
                )
                runtimeSetStatus("Setup code received")
                Task {
                    await saveBridgeProfile(
                        bridgeID: bridgeID,
                        name: name,
                        endpoint: serverEndpoint,
                        pairingCode: pairingCode ?? "",
                        existingAuthToken: nil,
                        libp2pPeerID: libp2pPeerID
                    )
                }
            }
        } catch {
            let message = normalizeErrorMessage(error)
            NativeHostDebugLog.write("import-error \(message)")
            renderConnectionFailure(
                profileName: "",
                endpoint: "",
                bridgeID: nil,
                message: message
            )
        }
    }

    private func saveBridgeProfile(
        bridgeID: String?,
        name: String,
        endpoint: String,
        pairingCode: String,
        existingAuthToken: String?,
        libp2pPeerID: String? = nil
    ) async {
        do {
            let normalizedEndpoint = BridgeAPI.normalizeEndpoint(endpoint)
            var authToken = existingAuthToken
            NativeHostDebugLog.write("save-bridge-profile name=\(name) endpoint=\(normalizedEndpoint) pairing=\(!pairingCode.isEmpty)")

            updateConnectionProgress(
                bridgeID: bridgeID,
                profileName: name,
                endpoint: normalizedEndpoint,
                stage: pairingCode.isEmpty ? .openingWorkspace : .pairingDevice,
                requiresPairing: !pairingCode.isEmpty
            )
            runtimeSetStatus("Preparing bridge connection")

            var loadTarget = try await resolveBridgeLoadTarget(
                endpoint: normalizedEndpoint,
                authToken: authToken
            )
            var connection = try await BridgeAPI.fetchConnectionTarget(
                baseURL: loadTarget.baseURL.absoluteString,
                authToken: loadTarget.usesLocalProxy ? nil : authToken
            )

            if !pairingCode.isEmpty {
                updateConnectionProgress(
                    bridgeID: bridgeID,
                    profileName: name,
                    endpoint: normalizedEndpoint,
                    stage: .pairingDevice,
                    requiresPairing: true
                )
                runtimeSetStatus("Approving this iPad")
                let pairing = try await BridgeAPI.completeDevicePairing(
                    baseURL: loadTarget.baseURL.absoluteString,
                    pairingCode: pairingCode,
                    authToken: loadTarget.usesLocalProxy ? nil : authToken
                )
                authToken = pairing.accessToken
                loadTarget = try await resolveBridgeLoadTarget(
                    endpoint: normalizedEndpoint,
                    authToken: authToken
                )
                connection = try await BridgeAPI.fetchConnectionTarget(
                    baseURL: loadTarget.baseURL.absoluteString,
                    authToken: loadTarget.usesLocalProxy ? nil : authToken
                )
            } else if connection.authMode == "device-token", authToken?.isEmpty != false {
                if let localAuthPage = connection.localAuthPage {
                    throw NativeHostBridgeError.requestFailed(
                        "This bridge needs a fresh enrollment QR. Open \(localAuthPage) on the bridge host."
                    )
                }
                throw NativeHostBridgeError.requestFailed("This bridge needs to be re-enrolled from the bridge host.")
            }

            let resolvedBridgeID = connection.bridgeID ?? bridgeID
            let recommendedEndpoint = BridgeAPI.normalizeEndpoint(connection.recommendedServerEndpoint)
            let existingProfile = profiles.first {
                matchesBridgeIdentity(profile: $0, bridgeID: resolvedBridgeID, endpoint: recommendedEndpoint) &&
                    ($0.name == name || BridgeAPI.normalizeEndpoint($0.serverEndpoint) == recommendedEndpoint)
            }
            let profile = BridgeProfile(
                id: existingProfile?.id ?? profileStore.createProfileID(name: name, endpoint: recommendedEndpoint, bridgeID: resolvedBridgeID),
                bridgeID: resolvedBridgeID,
                name: name,
                serverEndpoint: recommendedEndpoint,
                authToken: authToken,
                lastUsedAtMilliseconds: nil,
                libp2pPeerID: libp2pPeerID
            )
            profileStore.write(profile)
            refreshProfiles()

            updateConnectionProgress(
                bridgeID: resolvedBridgeID,
                profileName: profile.name,
                endpoint: profile.serverEndpoint,
                stage: .openingWorkspace,
                requiresPairing: !pairingCode.isEmpty
            )
            runtimeSetStatus("Opening your workspace")
            openBridge(profile)
        } catch {
            let message = normalizeErrorMessage(error)
            NativeHostDebugLog.write("save-bridge-profile-error endpoint=\(endpoint) message=\(message)")
            renderConnectionFailure(
                profileName: name,
                endpoint: BridgeAPI.normalizeEndpoint(endpoint),
                bridgeID: bridgeID,
                message: message
            )
        }
    }

    private func openBridge(_ profile: BridgeProfile, initialStatusMessage: String = "Opening your workspace") {
        NativeHostDebugLog.write("open-bridge profile=\(profile.id) endpoint=\(profile.serverEndpoint)")
        preferencesStore.setAutoResumeActiveSession(true)
        runtime.reset()
        activeLoadTarget = nil

        if currentConnectionStage == nil {
            updateConnectionProgress(
                bridgeID: profile.bridgeID,
                profileName: profile.name,
                endpoint: profile.serverEndpoint,
                stage: .openingWorkspace,
                requiresPairing: false
            )
        }

        activeProfile = profile
        refreshProfiles()
        chromeState = .loading
        runtimeSetStatus(initialStatusMessage)
        let generation = bridgeLoadGeneration + 1
        bridgeLoadGeneration = generation

        Task {
            do {
                let loadTarget = try await resolveBridgeLoadTarget(
                    endpoint: profile.serverEndpoint,
                    authToken: profile.authToken
                )
                do {
                    try await synchronizeMobileHostSettings(
                        profile: profile,
                        loadTarget: loadTarget
                    )
                } catch {
                    NativeHostDebugLog.write("mobile-host-settings-sync-error message=\(normalizeErrorMessage(error))")
                }
                NativeHostDebugLog.write("open-bridge-resolved-load-target baseURL=\(loadTarget.baseURL.absoluteString) localProxy=\(loadTarget.usesLocalProxy)")
                let bootstrapState = try? await runtime.hydrateBridgeBootstrapState(
                    loadTarget: loadTarget,
                    authToken: loadTarget.usesLocalProxy ? nil : profile.authToken
                )

                guard generation == bridgeLoadGeneration, activeProfile == profile else {
                    return
                }

                if let bootstrapState {
                    runtime.applyBridgeBootstrapState(bootstrapState)
                }
                activeLoadTarget = loadTarget

                await webBridge.configureCookies(
                    baseURL: loadTarget.baseURL,
                    authToken: profile.authToken,
                    usesLocalProxy: loadTarget.usesLocalProxy
                )

                guard let url = BridgeAPI.buildRemoteShellURL(
                    baseURL: loadTarget.baseURL.absoluteString,
                    theme: resolvedThemeName
                ) else {
                    throw NativeHostBridgeError.invalidURL("Failed to build the bridge UI URL.")
                }

                var headers: [String: String] = [:]
                if !loadTarget.usesLocalProxy, let authToken = profile.authToken, !authToken.isEmpty {
                    headers["Authorization"] = "Bearer \(authToken)"
                }
                webBridge.load(url: url, headers: headers)
            } catch {
                guard generation == bridgeLoadGeneration, activeProfile == profile else {
                    return
                }
                activeLoadTarget = nil
                let message = normalizeErrorMessage(error)
                NativeHostDebugLog.write("open-bridge-error message=\(message)")
                runtimeSetStatus(message)
                chromeState = .error
                scheduleAutoReconnectIfNeeded(for: profile)
            }
        }
    }

    private func synchronizeMobileHostSettings(profile: BridgeProfile, loadTarget: BridgeLoadTarget) async throws {
        _ = try await BridgeAPI.performDirectRPC(
            baseURL: loadTarget.baseURL.absoluteString,
            method: "set-configuration",
            params: [
                "key": Self.fullDesktopFileAccessConfigurationKey,
                "value": fullDesktopFileAccessEnabled,
            ],
            authToken: loadTarget.usesLocalProxy ? nil : profile.authToken
        )
    }

    private func refreshProfiles() {
        profiles = profileStore.list()
        runtime.syncSavedConnectionsState(
            profiles: profiles,
            activeProfileID: activeProfile?.id ?? profileStore.readActive()?.id
        )
    }

    private func updateConnectionProgress(
        bridgeID: String?,
        profileName: String,
        endpoint: String,
        stage: NativeHostConnectionStage,
        requiresPairing: Bool
    ) {
        currentConnectionStage = stage
        currentConnectionRequiresPairing = requiresPairing
        activeProfile = displayedProfile(
            bridgeID: bridgeID,
            profileName: profileName,
            endpoint: endpoint,
            transientID: "pending"
        )
        chromeState = .loading
    }

    private func renderConnectionFailure(
        profileName: String,
        endpoint: String,
        bridgeID: String?,
        message: String
    ) {
        NSLog("CodexMobile connection failure endpoint=%@ message=%@", endpoint, message)
        NativeHostDebugLog.write("connection-failure endpoint=\(endpoint) message=\(message)")
        resetAutoReconnectState()
        currentConnectionStage = nil
        currentConnectionRequiresPairing = false
        activeLoadTarget = nil
        activeProfile = displayedProfile(
            bridgeID: bridgeID,
            profileName: profileName,
            endpoint: endpoint,
            transientID: "pending-error"
        )
        statusMessage = message
        chromeState = .error
        refreshProfiles()
    }

    private func displayedProfile(
        bridgeID: String?,
        profileName: String,
        endpoint: String,
        transientID: String
    ) -> BridgeProfile {
        if let existingProfile = profiles.first(where: { matchesBridgeIdentity(profile: $0, bridgeID: bridgeID, endpoint: endpoint) }) {
            return existingProfile
        }
        return BridgeProfile(
            id: transientID,
            bridgeID: bridgeID?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            name: profileName.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Codex Mobile",
            serverEndpoint: endpoint.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Codex Mobile",
            authToken: nil,
            lastUsedAtMilliseconds: nil
        )
    }

    private func renderDisconnectedHome(message: String) {
        resetAutoReconnectState()
        currentConnectionStage = nil
        currentConnectionRequiresPairing = false
        chromeState = .disconnected
        statusMessage = message
        activeProfile = nil
        activeLoadTarget = nil
        refreshProfiles()
    }

    private func updateWebTheme() {
        guard isConnected else {
            return
        }
        let script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (host && typeof host.updateTheme === "function") {
                host.updateTheme(\(jsonEncodedString(resolvedThemeName)));
              }
            })();
            """
        webBridge.evaluateJavaScript(script)
    }

    private func reconnectDelayNanoseconds(for attempt: Int) -> UInt64 {
        switch attempt {
        case 1:
            return 1_200_000_000
        case 2:
            return 2_500_000_000
        case 3:
            return 5_000_000_000
        case 4:
            return 10_000_000_000
        default:
            return 15_000_000_000
        }
    }

    private func shouldAutoReconnect(profile: BridgeProfile, message: String) -> Bool {
        if profile.id == "pending-error" {
            return false
        }

        let normalized = message.trimmingCharacters(in: .whitespacesAndNewlines)
        if normalized.isEmpty {
            return true
        }

        return !(normalized == "This QR code has expired. Refresh the desktop QR code and try again." ||
            normalized == "This saved setup is no longer valid. Generate a fresh QR code on desktop and try again." ||
            normalized.localizedCaseInsensitiveContains("expired") ||
            normalized.localizedCaseInsensitiveContains("fresh enrollment qr") ||
            normalized.localizedCaseInsensitiveContains("re-enrolled") ||
            normalized.localizedCaseInsensitiveContains("no longer valid") ||
            normalized.localizedCaseInsensitiveContains("invalid key"))
    }

    private func isBridgeConnectionIssue(_ error: Error) -> Bool {
        if let bridgeError = error as? NativeHostBridgeError {
            switch bridgeError {
            case .webSocketClosed:
                return true
            default:
                break
            }
        }

        let nsError = error as NSError
        if nsError.domain == NSURLErrorDomain {
            switch nsError.code {
            case NSURLErrorTimedOut,
                 NSURLErrorCannotFindHost,
                 NSURLErrorCannotConnectToHost,
                 NSURLErrorNetworkConnectionLost,
                 NSURLErrorDNSLookupFailed,
                 NSURLErrorNotConnectedToInternet,
                 NSURLErrorInternationalRoamingOff,
                 NSURLErrorCallIsActive,
                 NSURLErrorDataNotAllowed:
                return true
            default:
                break
            }
        }

        let description = nsError.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        return description.localizedCaseInsensitiveContains("timed out") ||
            description.localizedCaseInsensitiveContains("not connected") ||
            description.localizedCaseInsensitiveContains("connection refused") ||
            description.localizedCaseInsensitiveContains("connection reset") ||
            description.localizedCaseInsensitiveContains("network is unreachable") ||
            description.localizedCaseInsensitiveContains("failed to connect") ||
            description.localizedCaseInsensitiveContains("websocket closed")
    }

    private func scheduleAutoReconnectIfNeeded(for profile: BridgeProfile) {
        guard preferencesStore.shouldAutoResumeActiveSession(),
              chromeState == .error,
              shouldAutoReconnect(profile: profile, message: statusMessage),
              autoReconnectTask == nil else {
            return
        }

        let nextAttempt = autoReconnectProfileID == profile.id ? autoReconnectAttemptCount + 1 : 1
        autoReconnectProfileID = profile.id
        autoReconnectAttemptCount = nextAttempt
        autoReconnectTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: self?.reconnectDelayNanoseconds(for: nextAttempt) ?? 1_200_000_000)
            guard !Task.isCancelled else {
                return
            }
            guard let self,
                  self.activeProfile?.id == profile.id,
                  self.chromeState == .error else {
                return
            }
            self.autoReconnectTask = nil
            self.openBridge(profile, initialStatusMessage: "Trying to restore this session")
        }
    }

    private func resetAutoReconnectState() {
        autoReconnectTask?.cancel()
        autoReconnectTask = nil
        autoReconnectProfileID = nil
        autoReconnectAttemptCount = 0
    }

    private func resolveBridgeLoadTarget(
        endpoint: String,
        authToken: String?
    ) async throws -> BridgeLoadTarget {
        guard let directURL = URL(string: BridgeAPI.deriveServerHTTPBaseURL(endpoint)) else {
            NativeHostDebugLog.write("resolve-load-target invalid-url endpoint=\(endpoint)")
            throw NativeHostBridgeError.invalidURL("Invalid bridge endpoint: \(endpoint)")
        }
        _ = authToken
        NativeHostDebugLog.write("resolve-load-target direct baseURL=\(directURL.absoluteString)")
        return BridgeLoadTarget(baseURL: directURL, usesLocalProxy: false)
    }

    private func matchesBridgeIdentity(profile: BridgeProfile, bridgeID: String?, endpoint: String) -> Bool {
        let normalizedBridgeID = bridgeID?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !normalizedBridgeID.isEmpty {
            return profile.bridgeID?.trimmingCharacters(in: .whitespacesAndNewlines) == normalizedBridgeID
        }
        return BridgeAPI.normalizeEndpoint(profile.serverEndpoint) == BridgeAPI.normalizeEndpoint(endpoint)
    }

    func normalizeErrorMessage(_ error: Error) -> String {
        if let bridgeError = error as? NativeHostBridgeError,
           let description = bridgeError.errorDescription {
            return description
        }

        let nsError = error as NSError
        let description = nsError.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        if description.localizedCaseInsensitiveContains("timed out") {
            return "The bridge did not respond in time. Check that the desktop session is still reachable from this iPad."
        }
        if nsError.domain == NSURLErrorDomain {
            switch nsError.code {
            case NSURLErrorNotConnectedToInternet, NSURLErrorNetworkConnectionLost:
                return "This iPad could not reach the bridge. Make sure the desktop exposure is reachable from the device."
            case NSURLErrorCannotFindHost, NSURLErrorCannotConnectToHost:
                return "The bridge host could not be reached from this iPad."
            default:
                break
            }
        }
        return description.isEmpty ? "Something went wrong while connecting to the bridge." : description
    }

    func loadBrowserAttachTargets() async throws -> [NativeHostBrowserAttachTarget] {
        let result = try await performBrowserSessionRPC(method: "browser-session/list-targets", params: [:])
        return ((result["targets"] as? [Any]) ?? []).compactMap { rawTarget in
            guard let target = rawTarget as? JSONDictionary,
                  let targetID = (target["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                  let debugBaseURL = (target["debugBaseURL"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty else {
                return nil
            }
            let title = (target["title"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Untitled Tab"
            let url = (target["url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? ""
            return NativeHostBrowserAttachTarget(
                id: targetID,
                title: title,
                url: url,
                debugBaseURL: debugBaseURL
            )
        }
    }

    func startPreviewBrowserSession(
        url: String,
        preset: NativeHostBrowserViewportPreset
    ) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/start",
            params: [
                "source": "preview",
                "url": url,
                "preset": preset.rawValue,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func startAttachedBrowserSession(
        target: NativeHostBrowserAttachTarget,
        preset: NativeHostBrowserViewportPreset
    ) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/start",
            params: [
                "source": "attach",
                "targetId": target.id,
                "debugBaseURL": target.debugBaseURL,
                "preset": preset.rawValue,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func fetchBrowserSessionSnapshot(sessionID: String) async throws -> NativeHostBrowserSessionSnapshot {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/snapshot",
            params: [
                "sessionId": sessionID,
                "quality": 55,
            ]
        )
        return try parseBrowserSessionSnapshot(from: result)
    }

    func fetchBrowserSessionStatus(sessionID: String) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/status",
            params: [
                "sessionId": sessionID,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func waitForNextBrowserSessionSnapshot(
        sessionID: String,
        afterRevision: Int
    ) async throws -> NativeHostBrowserSessionSnapshot {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/next-frame",
            params: [
                "sessionId": sessionID,
                "afterRevision": afterRevision,
                "quality": 55,
                "timeoutMs": 8000,
            ]
        )
        return try parseBrowserSessionSnapshot(from: result)
    }

    func navigateBrowserSession(
        sessionID: String,
        action: String,
        url: String? = nil
    ) async throws -> NativeHostBrowserSessionSummary {
        var params: JSONDictionary = [
            "sessionId": sessionID,
            "action": action,
        ]
        if let url = url?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty {
            params["url"] = url
        }
        let result = try await performBrowserSessionRPC(
            method: "browser-session/navigate",
            params: params
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionTap(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "tap",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionPointerDown(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerDown",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionPointerMove(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerMove",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionPointerUp(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerUp",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionScroll(
        sessionID: String,
        deltaX: Double = 0,
        deltaY: Double,
        xNorm: Double = 0.5,
        yNorm: Double = 0.5
    ) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "scroll",
                "deltaX": deltaX,
                "deltaY": deltaY,
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionText(sessionID: String, text: String) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "text",
                "text": text,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionTextState(
        sessionID: String,
        text: String,
        selectionStart: Int,
        selectionEnd: Int
    ) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "textState",
                "text": text,
                "selectionStart": selectionStart,
                "selectionEnd": selectionEnd,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func sendBrowserSessionKey(sessionID: String, key: String) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/input",
            params: [
                "sessionId": sessionID,
                "type": "key",
                "key": key,
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func syncBrowserSessionEditableState(
        sessionID: String,
        delayMilliseconds: Int = 0
    ) async throws -> NativeHostBrowserSessionSummary {
        let result = try await performBrowserSessionRPC(
            method: "browser-session/sync-editable",
            params: [
                "sessionId": sessionID,
                "delayMs": max(delayMilliseconds, 0),
            ]
        )
        return try parseBrowserSessionSummary(from: result)
    }

    func browserSessionStreamConnection(sessionID: String) async throws -> (url: URL, headers: [String: String]) {
        guard let activeProfile else {
            throw NativeHostBridgeError.requestFailed("Open a desktop session before starting Browser.")
        }
        let loadTarget: BridgeLoadTarget
        if let activeLoadTarget {
            loadTarget = activeLoadTarget
        } else {
            loadTarget = try await resolveBridgeLoadTarget(
                endpoint: activeProfile.serverEndpoint,
                authToken: activeProfile.authToken
            )
        }

        var components = URLComponents(url: loadTarget.baseURL, resolvingAgainstBaseURL: false) ?? URLComponents()
        components.scheme = components.scheme == "https" ? "wss" : "ws"
        components.path = "/codex-mobile/browser-session/stream"
        components.queryItems = [URLQueryItem(name: "sessionId", value: sessionID)]
        guard let streamURL = components.url else {
            throw NativeHostBridgeError.invalidURL("Failed to create the browser frame stream URL.")
        }

        let headers: [String: String]
        if loadTarget.usesLocalProxy || activeProfile.authToken?.isEmpty != false {
            headers = [:]
        } else {
            headers = ["Authorization": "Bearer \(activeProfile.authToken!)"]
        }
        return (streamURL, headers)
    }

    func decodeBrowserSessionSummary(from payload: JSONDictionary) throws -> NativeHostBrowserSessionSummary {
        try parseBrowserSessionSummary(from: payload)
    }

    func startDesktopSession() async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(method: "desktop-session/start", params: [:])
        return try parseDesktopSessionSummary(from: result)
    }

    func fetchDesktopSessionStatus(sessionID: String) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/status",
            params: ["sessionId": sessionID]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionTap(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "tap",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionPointerDown(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerDown",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionPointerMove(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerMove",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionPointerUp(sessionID: String, xNorm: Double, yNorm: Double) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "pointerUp",
                "xNorm": xNorm,
                "yNorm": yNorm,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionText(sessionID: String, text: String) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "text",
                "text": text,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionTextState(
        sessionID: String,
        text: String,
        selectionStart: Int,
        selectionEnd: Int
    ) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "textState",
                "text": text,
                "selectionStart": selectionStart,
                "selectionEnd": selectionEnd,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionKey(
        sessionID: String,
        key: String,
        modifiers: [String] = []
    ) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "key",
                "key": key,
                "modifiers": modifiers,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func sendDesktopSessionScroll(sessionID: String, deltaY: Double) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/input",
            params: [
                "sessionId": sessionID,
                "type": "scroll",
                "deltaY": deltaY,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func stopDesktopSession(sessionID: String) async {
        _ = try? await performDesktopSessionRPC(
            method: "desktop-session/stop",
            params: ["sessionId": sessionID]
        )
    }

    func createDesktopSessionWebRTCOffer(sessionID: String) async throws -> NativeHostDesktopWebRTCOffer {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/webrtc-offer",
            params: ["sessionId": sessionID]
        )
        guard let peerID = (result["peerId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty else {
            throw NativeHostBridgeError.invalidResponse("Desktop WebRTC offer did not include a peer id.")
        }
        guard let sdp = result["sdp"] as? String, sdp.isEmpty == false else {
            throw NativeHostBridgeError.invalidResponse("Desktop WebRTC offer did not include an SDP offer.")
        }
        return NativeHostDesktopWebRTCOffer(peerID: peerID, sdp: sdp)
    }

    func submitDesktopSessionWebRTCAnswer(sessionID: String, peerID: String, sdp: String) async throws -> NativeHostDesktopSessionSummary {
        let result = try await performDesktopSessionRPC(
            method: "desktop-session/webrtc-answer",
            params: [
                "sessionId": sessionID,
                "peerId": peerID,
                "sdp": sdp,
            ]
        )
        return try parseDesktopSessionSummary(from: result)
    }

    func decodeDesktopSessionSummary(from payload: JSONDictionary) throws -> NativeHostDesktopSessionSummary {
        try parseDesktopSessionSummary(from: payload)
    }

    func stopBrowserSession(sessionID: String) async {
        _ = try? await performBrowserSessionRPC(
            method: "browser-session/stop",
            params: ["sessionId": sessionID]
        )
    }

    private func performBrowserSessionRPC(method: String, params: JSONDictionary) async throws -> JSONDictionary {
        guard let activeProfile else {
            throw NativeHostBridgeError.requestFailed("Open a desktop session before starting Browser.")
        }
        let loadTarget: BridgeLoadTarget
        if let activeLoadTarget {
            loadTarget = activeLoadTarget
        } else {
            loadTarget = try await resolveBridgeLoadTarget(
                endpoint: activeProfile.serverEndpoint,
                authToken: activeProfile.authToken
            )
        }

        let result = try await BridgeAPI.performDirectRPC(
            baseURL: loadTarget.baseURL.absoluteString,
            method: method,
            params: params,
            authToken: loadTarget.usesLocalProxy ? nil : activeProfile.authToken
        )
        if let errorMessage = (result["error"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty {
            throw NativeHostBridgeError.requestFailed(errorMessage)
        }
        return result
    }

    private func performDesktopSessionRPC(method: String, params: JSONDictionary) async throws -> JSONDictionary {
        guard let activeProfile else {
            throw NativeHostBridgeError.requestFailed("Open a desktop session before starting Desktop.")
        }
        let loadTarget: BridgeLoadTarget
        if let activeLoadTarget {
            loadTarget = activeLoadTarget
        } else {
            loadTarget = try await resolveBridgeLoadTarget(
                endpoint: activeProfile.serverEndpoint,
                authToken: activeProfile.authToken
            )
        }

        let result = try await BridgeAPI.performDirectRPC(
            baseURL: loadTarget.baseURL.absoluteString,
            method: method,
            params: params,
            authToken: loadTarget.usesLocalProxy ? nil : activeProfile.authToken
        )
        if let errorMessage = (result["error"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty {
            throw NativeHostBridgeError.requestFailed(errorMessage)
        }
        return result
    }

    private func parseBrowserSessionSummary(from payload: JSONDictionary) throws -> NativeHostBrowserSessionSummary {
        guard let sessionID = (payload["sessionId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty else {
            throw NativeHostBridgeError.invalidResponse("Browser session response did not include a session id.")
        }
        let source = NativeHostBrowserSessionSource(
            rawValue: (payload["source"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        ) ?? .preview
        let viewport = payload["viewport"] as? JSONDictionary ?? [:]
        let width = Int((viewport["width"] as? NSNumber)?.doubleValue ?? (payload["width"] as? NSNumber)?.doubleValue ?? 0)
        let height = Int((viewport["height"] as? NSNumber)?.doubleValue ?? (payload["height"] as? NSNumber)?.doubleValue ?? 0)
        let scale = (viewport["scale"] as? NSNumber)?.doubleValue ?? 1
        return NativeHostBrowserSessionSummary(
            sessionID: sessionID,
            source: source,
            title: (payload["title"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Browser Session",
            url: (payload["url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "",
            loading: payload["loading"] as? Bool ?? false,
            isStreaming: payload["streaming"] as? Bool ?? false,
            textInputActive: payload["textInputActive"] as? Bool ?? false,
            editableText: payload["editableText"] as? String ?? "",
            selectionStart: Int((payload["selectionStart"] as? NSNumber)?.doubleValue ?? 0),
            selectionEnd: Int((payload["selectionEnd"] as? NSNumber)?.doubleValue ?? 0),
            canGoBack: payload["canGoBack"] as? Bool ?? false,
            canGoForward: payload["canGoForward"] as? Bool ?? false,
            width: max(width, 1),
            height: max(height, 1),
            scale: max(scale, 1),
            debugBaseURL: (payload["debugBaseURL"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            targetID: (payload["targetId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
        )
    }

    private func parseBrowserSessionSnapshot(from payload: JSONDictionary) throws -> NativeHostBrowserSessionSnapshot {
        let summary = try parseBrowserSessionSummary(from: payload)
        guard let imageBase64 = (payload["imageBase64"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
              let imageData = Data(base64Encoded: imageBase64) else {
            throw NativeHostBridgeError.invalidResponse("Browser session snapshot did not include valid image data.")
        }
        let console = payload["console"] as? JSONDictionary ?? [:]
        let network = payload["network"] as? JSONDictionary ?? [:]
        let revision = Int((payload["revision"] as? NSNumber)?.doubleValue ?? 0)
        return NativeHostBrowserSessionSnapshot(
            summary: summary,
            revision: revision,
            mimeType: (payload["mimeType"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "image/jpeg",
            imageData: imageData,
            consoleErrorCount: Int((console["errorCount"] as? NSNumber)?.doubleValue ?? 0),
            consoleWarnCount: Int((console["warnCount"] as? NSNumber)?.doubleValue ?? 0),
            consoleLines: (console["lastLines"] as? [String]) ?? [],
            networkInflightCount: Int((network["inflight"] as? NSNumber)?.doubleValue ?? 0),
            networkFailedCount: Int((network["failedCount"] as? NSNumber)?.doubleValue ?? 0),
            networkFailedLines: (network["lastFailed"] as? [String]) ?? []
        )
    }

    private func parseDesktopSessionSummary(from payload: JSONDictionary) throws -> NativeHostDesktopSessionSummary {
        guard let sessionID = (payload["sessionId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty else {
            throw NativeHostBridgeError.invalidResponse("Desktop session response did not include a session id.")
        }
        let width = Int((payload["width"] as? NSNumber)?.doubleValue ?? 0)
        let height = Int((payload["height"] as? NSNumber)?.doubleValue ?? 0)
        let scale = (payload["scale"] as? NSNumber)?.doubleValue ?? 1
        return NativeHostDesktopSessionSummary(
            sessionID: sessionID,
            title: (payload["title"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Desktop",
            isStreaming: payload["streaming"] as? Bool ?? false,
            width: max(width, 0),
            height: max(height, 0),
            scale: max(scale, 1),
            preferredTransport: (payload["preferredTransport"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "webrtc",
            videoCodec: (payload["videoCodec"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            videoReady: payload["videoReady"] as? Bool ?? false,
            videoError: (payload["videoError"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            lastError: (payload["lastError"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            textInputActive: payload["textInputActive"] as? Bool ?? false,
            editableText: payload["editableText"] as? String ?? "",
            editablePlaceholder: payload["editablePlaceholder"] as? String ?? "",
            editableRole: (payload["editableRole"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            selectionStart: Int((payload["selectionStart"] as? NSNumber)?.doubleValue ?? 0),
            selectionEnd: Int((payload["selectionEnd"] as? NSNumber)?.doubleValue ?? 0)
        )
    }

}

extension NativeHostModel: NativeHostBridgeRuntimeDelegate {
    var runtimeActiveProfile: BridgeProfile? { activeProfile }
    var runtimeActiveLoadTarget: BridgeLoadTarget? { activeLoadTarget }

    func runtimeResolveBridgeLoadTarget(for profile: BridgeProfile) async throws -> BridgeLoadTarget {
        try await resolveBridgeLoadTarget(
            endpoint: profile.serverEndpoint,
            authToken: profile.authToken
        )
    }

    func runtimeSetStatus(_ message: String) {
        statusMessage = message
    }

    func runtimeNormalizeError(_ error: Error) -> String {
        normalizeErrorMessage(error)
    }

    func runtimeHandleBridgeConnectionIssue(_ error: Error) {
        guard isBridgeConnectionIssue(error),
              let activeProfile,
              chromeState != .disconnected,
              chromeState != .error else {
            return
        }

        NativeHostDebugLog.write("bridge-connection-lost error=\(normalizeErrorMessage(error))")
        activeLoadTarget = nil
        statusMessage = "Lost contact with the desktop bridge. Reconnecting."
        chromeState = .error
        scheduleAutoReconnectIfNeeded(for: activeProfile)
    }

    func runtimeOpenExternalURL(_ url: URL) {
        UIApplication.shared.open(url)
    }

    func runtimePickFiles(title: String) async throws -> NativeHostAttachmentSelection {
        let normalizedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Add Photos & Files"
        if let presentingController = webBridge.currentPresentingViewController() {
            NativeHostDebugLog.write("pick-files presenter=webview:\(String(describing: type(of: presentingController))) title=\(normalizedTitle)")
            let picker = makeAttachmentPicker(
                presentingController: presentingController,
                allowsDesktopFiles: true
            )
            return try await picker.pick(title: normalizedTitle)
        }
        guard let presentingController = topPresentingViewController() else {
            NativeHostDebugLog.write("pick-files presenter-missing title=\(normalizedTitle)")
            throw NativeHostBridgeError.requestFailed("Codex Mobile could not present the file picker.")
        }
        NativeHostDebugLog.write("pick-files presenter=scene:\(String(describing: type(of: presentingController))) title=\(normalizedTitle)")
        let picker = makeAttachmentPicker(
            presentingController: presentingController,
            allowsDesktopFiles: true
        )
        return try await picker.pick(title: normalizedTitle)
    }
}

extension NativeHostModel: NativeHostWebBridgeDelegate {
    func webBridgeDidReceive(message: String) {
        runtime.handleEnvelope(message)
    }

    func webBridgeDidStartNavigation(url: URL?) {
        guard chromeState != .connected else {
            return
        }
        chromeState = .loading
    }

    func webBridgeDidFinishNavigation(url: URL?) {
        guard let activeProfile,
              let url,
              url.absoluteString != "about:blank" else {
            return
        }
        NativeHostDebugLog.write("webview-finished url=\(url.absoluteString)")
        chromeState = .connected
        statusMessage = "Connected to \(activeProfile.serverEndpoint)"
        currentConnectionStage = nil
        resetAutoReconnectState()
        updateWebTheme()
    }

    func webBridgeDidFailNavigation(message: String) {
        guard let activeProfile else {
            return
        }
        NativeHostDebugLog.write("webview-failed message=\(message)")
        statusMessage = message
        chromeState = .error
        scheduleAutoReconnectIfNeeded(for: activeProfile)
    }

    func webBridgeCurrentBaseURL() -> URL? {
        activeLoadTarget?.baseURL
    }

    func webBridgeOpenExternalURL(_ url: URL) {
        UIApplication.shared.open(url)
    }

    func webBridgePickOpenPanelFiles(title: String, allowsMultipleSelection: Bool) async throws -> [URL] {
        let normalizedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Add Photos & Files"
        if let presentingController = webBridge.currentPresentingViewController() {
            NativeHostDebugLog.write("open-panel presenter=webview:\(String(describing: type(of: presentingController))) title=\(normalizedTitle) multiple=\(allowsMultipleSelection)")
            let picker = makeAttachmentPicker(
                presentingController: presentingController,
                allowsDesktopFiles: false
            )
            return try await picker.pickTemporaryFileURLs(
                title: normalizedTitle,
                allowsMultipleSelection: allowsMultipleSelection
            )
        }
        guard let presentingController = topPresentingViewController() else {
            NativeHostDebugLog.write("open-panel presenter-missing title=\(normalizedTitle) multiple=\(allowsMultipleSelection)")
            throw NativeHostBridgeError.requestFailed("Codex Mobile could not present the file picker.")
        }
        NativeHostDebugLog.write("open-panel presenter=scene:\(String(describing: type(of: presentingController))) title=\(normalizedTitle) multiple=\(allowsMultipleSelection)")
        let picker = makeAttachmentPicker(
            presentingController: presentingController,
            allowsDesktopFiles: false
        )
        return try await picker.pickTemporaryFileURLs(
            title: normalizedTitle,
            allowsMultipleSelection: allowsMultipleSelection
        )
    }
}

private extension NativeHostModel {
    func makeAttachmentPicker(
        presentingController: UIViewController,
        allowsDesktopFiles: Bool
    ) -> NativeHostAttachmentPicker {
        let desktopPicker: ((String) async throws -> [NativeHostDesktopFileReference])?
        if allowsDesktopFiles, fullDesktopFileAccessEnabled {
            desktopPicker = { [weak self, weak presentingController] title in
                guard let self else {
                    throw NativeHostBridgeError.requestFailed("Codex Mobile lost the attachment picker context.")
                }
                guard let presentingController else {
                    throw NativeHostBridgeError.requestFailed("Codex Mobile could not present the desktop file browser.")
                }
                return try await self.presentDesktopFileBrowser(
                    title: title,
                    presentingController: presentingController
                )
            }
        } else {
            desktopPicker = nil
        }

        return NativeHostAttachmentPicker(
            presentingController: presentingController,
            desktopPicker: desktopPicker
        )
    }

    func presentDesktopFileBrowser(
        title: String,
        presentingController: UIViewController
    ) async throws -> [NativeHostDesktopFileReference] {
        guard fullDesktopFileAccessEnabled else {
            throw NativeHostBridgeError.requestFailed("Enable Full Desktop File Access in Settings before browsing desktop files.")
        }
        let browser = NativeHostDesktopFileBrowser(
            presentingController: presentingController,
            title: title,
            recentDirectoriesProvider: { [preferencesStore = self.preferencesStore] in
                preferencesStore.readRecentDesktopDirectories()
            },
            recentDirectoriesSaver: { [preferencesStore = self.preferencesStore] paths in
                preferencesStore.writeRecentDesktopDirectories(paths)
            },
            thumbnailLoader: { [weak self] entry in
                guard let self else {
                    return nil
                }
                return await self.loadDesktopThumbnailData(for: entry)
            }
        ) { [weak self] directoryPath in
            guard let self else {
                throw NativeHostBridgeError.requestFailed("Codex Mobile lost the desktop browser context.")
            }
            return try await self.loadDesktopDirectoryListing(directoryPath: directoryPath)
        }
        return try await browser.pickFiles()
    }

    func loadDesktopDirectoryListing(directoryPath: String?) async throws -> NativeHostDesktopDirectoryListing {
        guard let activeProfile else {
            throw NativeHostBridgeError.requestFailed("No active desktop session is available.")
        }
        let loadTarget: BridgeLoadTarget
        if let activeLoadTarget {
            loadTarget = activeLoadTarget
        } else {
            loadTarget = try await resolveBridgeLoadTarget(
                endpoint: activeProfile.serverEndpoint,
                authToken: activeProfile.authToken
            )
        }
        try await synchronizeMobileHostSettings(
            profile: activeProfile,
            loadTarget: loadTarget
        )

        let result = try await BridgeAPI.performDirectRPC(
            baseURL: loadTarget.baseURL.absoluteString,
            method: "remote-workspace-directory-entries",
            params: [
                "directoryPath": directoryPath ?? "",
                "directoriesOnly": false,
            ],
            authToken: loadTarget.usesLocalProxy ? nil : activeProfile.authToken
        )

        if let errorMessage = (result["error"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines),
           !errorMessage.isEmpty {
            throw NativeHostBridgeError.requestFailed(errorMessage)
        }

        let resolvedDirectoryPath = (result["directoryPath"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty
        guard let resolvedDirectoryPath else {
            throw NativeHostBridgeError.invalidResponse("Desktop directory listing did not return a directory path.")
        }

        let entries = ((result["entries"] as? [Any]) ?? []).compactMap { rawEntry -> NativeHostDesktopDirectoryEntry? in
            guard let entry = rawEntry as? JSONDictionary else {
                return nil
            }
            guard let name = (entry["name"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty,
                let path = (entry["path"] as? String)?
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                    .nilIfEmpty else {
                return nil
            }
            return NativeHostDesktopDirectoryEntry(
                name: name,
                path: path,
                isDirectory: entry["isDirectory"] as? Bool ?? false
            )
        }

        return NativeHostDesktopDirectoryListing(
            directoryPath: resolvedDirectoryPath,
            entries: entries
        )
    }

    func loadDesktopThumbnailData(for entry: NativeHostDesktopDirectoryEntry) async -> Data? {
        guard entry.isDirectory == false,
              desktopPathSupportsThumbnail(entry.path),
              let activeProfile else {
            return nil
        }

        let cacheKey = activeProfile.id + "|" + entry.path
        return await desktopThumbnailCache.data(for: cacheKey) { [weak self] in
            guard let self else {
                return nil
            }
            return await self.fetchDesktopThumbnailData(for: entry, profile: activeProfile)
        }
    }

    private func fetchDesktopThumbnailData(
        for entry: NativeHostDesktopDirectoryEntry,
        profile: BridgeProfile
    ) async -> Data? {
        do {
            let loadTarget: BridgeLoadTarget
            if let activeLoadTarget {
                loadTarget = activeLoadTarget
            } else {
                loadTarget = try await resolveBridgeLoadTarget(
                    endpoint: profile.serverEndpoint,
                    authToken: profile.authToken
                )
            }

            let result = try await BridgeAPI.performDirectRPC(
                baseURL: loadTarget.baseURL.absoluteString,
                method: "resource/resolve",
                params: [
                    "path": entry.path,
                ],
                authToken: loadTarget.usesLocalProxy ? nil : profile.authToken
            )

            guard result["supported"] as? Bool == true,
                  let resourcePath = (result["url"] as? String)?
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                    .nilIfEmpty,
                  let resourceURL = URL(string: resourcePath, relativeTo: loadTarget.baseURL)?.absoluteURL else {
                return nil
            }

            var request = URLRequest(url: resourceURL, timeoutInterval: 15)
            if loadTarget.usesLocalProxy == false,
               let authToken = profile.authToken?.trimmingCharacters(in: .whitespacesAndNewlines),
               authToken.isEmpty == false {
                request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
            }

            let (data, response) = try await URLSession.shared.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse,
                  (200...299).contains(httpResponse.statusCode) else {
                return nil
            }

            return downsampleDesktopThumbnailData(from: data)
        } catch {
            NativeHostDebugLog.write("desktop-thumbnail-load-failed path=\(entry.path) message=\(normalizeErrorMessage(error))")
            return nil
        }
    }

    private func desktopPathSupportsThumbnail(_ path: String) -> Bool {
        let fileExtension = URL(fileURLWithPath: path).pathExtension.trimmingCharacters(in: .whitespacesAndNewlines)
        guard fileExtension.isEmpty == false,
              let type = UTType(filenameExtension: fileExtension) else {
            return false
        }
        return type.conforms(to: .image)
    }

    private func downsampleDesktopThumbnailData(from data: Data) -> Data? {
        let sourceOptions: CFDictionary = [
            kCGImageSourceShouldCache: false,
        ] as CFDictionary
        guard let imageSource = CGImageSourceCreateWithData(data as CFData, sourceOptions) else {
            return nil
        }

        let thumbnailOptions: CFDictionary = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: false,
            kCGImageSourceThumbnailMaxPixelSize: 160,
        ] as CFDictionary
        guard let cgImage = CGImageSourceCreateThumbnailAtIndex(imageSource, 0, thumbnailOptions) else {
            return nil
        }

        let image = UIImage(cgImage: cgImage)
        return image.pngData() ?? image.jpegData(compressionQuality: 0.75)
    }

    func topPresentingViewController() -> UIViewController? {
        let activeScenes = UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .filter { scene in
                scene.activationState == .foregroundActive || scene.activationState == .foregroundInactive
            }

        for scene in activeScenes {
            if let rootController = scene.windows.first(where: \.isKeyWindow)?.rootViewController {
                return rootController.topMostPresentedController()
            }
            if let rootController = scene.windows.first?.rootViewController {
                return rootController.topMostPresentedController()
            }
        }

        return nil
    }
}

@MainActor
final class NativeHostAttachmentPicker: NSObject, PHPickerViewControllerDelegate, UIDocumentPickerDelegate {
    private weak var presentingController: UIViewController?
    private let desktopPicker: ((String) async throws -> [NativeHostDesktopFileReference])?
    private var attachmentContinuation: CheckedContinuation<NativeHostAttachmentSelection, Error>?
    private var deviceFilesContinuation: CheckedContinuation<[NativeHostPickedFile], Error>?

    init(
        presentingController: UIViewController,
        desktopPicker: ((String) async throws -> [NativeHostDesktopFileReference])? = nil
    ) {
        self.presentingController = presentingController
        self.desktopPicker = desktopPicker
    }

    func pick(title: String) async throws -> NativeHostAttachmentSelection {
        try await withCheckedThrowingContinuation { continuation in
            attachmentContinuation = continuation
            NativeHostDebugLog.write("attachment-picker prompt title=\(normalizedPromptTitle(title)) desktop=\(desktopPicker != nil)")
            presentSourcePrompt(title: normalizedPromptTitle(title), includesDesktopFiles: desktopPicker != nil)
        }
    }

    private func pickDeviceFiles(title: String) async throws -> [NativeHostPickedFile] {
        try await withCheckedThrowingContinuation { continuation in
            deviceFilesContinuation = continuation
            NativeHostDebugLog.write("attachment-picker device-only prompt title=\(normalizedPromptTitle(title))")
            presentSourcePrompt(title: normalizedPromptTitle(title), includesDesktopFiles: false)
        }
    }

    func pickTemporaryFileURLs(title: String, allowsMultipleSelection: Bool) async throws -> [URL] {
        let files = try await pickDeviceFiles(title: title)
        let selectedFiles = allowsMultipleSelection ? files : Array(files.prefix(1))
        let urls = try persistPickedFilesToTemporaryURLs(selectedFiles)
        NativeHostDebugLog.write("attachment-picker open-panel-selected count=\(urls.count)")
        return urls
    }

    func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
        picker.dismiss(animated: true)
        NativeHostDebugLog.write("attachment-picker photos-picked count=\(results.count)")
        Task { @MainActor in
            do {
                finishDeviceFiles(try await loadPhotoResults(results))
            } catch {
                fail(with: error)
            }
        }
    }

    func documentPickerWasCancelled(_ controller: UIDocumentPickerViewController) {
        controller.dismiss(animated: true)
        NativeHostDebugLog.write("attachment-picker documents-cancelled")
        finishDeviceFiles([])
    }

    func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
        controller.dismiss(animated: true)
        NativeHostDebugLog.write("attachment-picker documents-picked count=\(urls.count)")
        Task { @MainActor in
            do {
                finishDeviceFiles(try await loadDocumentResults(urls))
            } catch {
                fail(with: error)
            }
        }
    }

    private func presentSourcePrompt(title: String, includesDesktopFiles: Bool) {
        guard let presentingController else {
            NativeHostDebugLog.write("attachment-picker presenter-missing title=\(title)")
            finishDeviceFiles([])
            return
        }

        let alert = UIAlertController(title: title, message: nil, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "Photos", style: .default) { [weak self] _ in
            NativeHostDebugLog.write("attachment-picker source=photos")
            Task { @MainActor [weak self] in
                self?.presentPhotosPicker()
            }
        })
        alert.addAction(UIAlertAction(title: "Files", style: .default) { [weak self] _ in
            NativeHostDebugLog.write("attachment-picker source=files")
            Task { @MainActor [weak self] in
                self?.presentDocumentPicker()
            }
        })
        if includesDesktopFiles {
            alert.addAction(UIAlertAction(title: "Desktop", style: .default) { [weak self] _ in
                NativeHostDebugLog.write("attachment-picker source=desktop")
                Task { @MainActor [weak self] in
                    await self?.presentDesktopPicker(title: title)
                }
            })
        }
        alert.addAction(UIAlertAction(title: "Cancel", style: .cancel) { [weak self] _ in
            NativeHostDebugLog.write("attachment-picker source=cancel")
            self?.finishDeviceFiles([])
        })
        presentingController.present(alert, animated: true)
    }

    private func presentPhotosPicker() {
        guard let presentingController else {
            NativeHostDebugLog.write("attachment-picker photos-presenter-missing")
            finishDeviceFiles([])
            return
        }

        var configuration = PHPickerConfiguration(photoLibrary: .shared())
        configuration.selectionLimit = 0
        configuration.filter = .images
        let picker = PHPickerViewController(configuration: configuration)
        picker.delegate = self
        NativeHostDebugLog.write("attachment-picker present-photos")
        presentingController.present(picker, animated: true)
    }

    private func presentDocumentPicker() {
        guard let presentingController else {
            NativeHostDebugLog.write("attachment-picker documents-presenter-missing")
            finishDeviceFiles([])
            return
        }

        let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.item], asCopy: true)
        picker.delegate = self
        picker.allowsMultipleSelection = true
        picker.title = "Select Files"
        NativeHostDebugLog.write("attachment-picker present-documents")
        presentingController.present(picker, animated: true)
    }

    private func presentDesktopPicker(title: String) async {
        guard let desktopPicker else {
            finishDeviceFiles([])
            return
        }
        do {
            let references = try await desktopPicker(title)
            finish(with: .desktopFiles(references))
        } catch {
            fail(with: error)
        }
    }

    private func finish(with selection: NativeHostAttachmentSelection) {
        if let continuation = attachmentContinuation {
            attachmentContinuation = nil
            deviceFilesContinuation = nil
            let count: Int
            switch selection {
            case let .deviceFiles(files):
                count = files.count
            case let .desktopFiles(references):
                count = references.count
            }
            NativeHostDebugLog.write("attachment-picker finish selection-count=\(count)")
            continuation.resume(returning: selection)
            return
        }

        switch selection {
        case let .deviceFiles(files):
            finishDeviceFiles(files)
        case .desktopFiles:
            finishDeviceFiles([])
        }
    }

    private func finishDeviceFiles(_ files: [NativeHostPickedFile]) {
        if let continuation = deviceFilesContinuation {
            deviceFilesContinuation = nil
            attachmentContinuation = nil
            NativeHostDebugLog.write("attachment-picker finish device-count=\(files.count)")
            continuation.resume(returning: files)
            return
        }
        if let continuation = attachmentContinuation {
            attachmentContinuation = nil
            NativeHostDebugLog.write("attachment-picker finish selection-count=\(files.count)")
            continuation.resume(returning: .deviceFiles(files))
        }
    }

    private func fail(with error: Error) {
        if let continuation = attachmentContinuation {
            attachmentContinuation = nil
            deviceFilesContinuation = nil
            NativeHostDebugLog.write("attachment-picker fail message=\((error as NSError).localizedDescription)")
            continuation.resume(throwing: error)
            return
        }
        guard let continuation = deviceFilesContinuation else {
            return
        }
        deviceFilesContinuation = nil
        NativeHostDebugLog.write("attachment-picker fail message=\((error as NSError).localizedDescription)")
        continuation.resume(throwing: error)
    }

    private func loadPhotoResults(_ results: [PHPickerResult]) async throws -> [NativeHostPickedFile] {
        var files: [NativeHostPickedFile] = []
        for result in results {
            if let file = try await loadPhotoResult(result) {
                files.append(file)
            }
        }
        return files
    }

    private func loadPhotoResult(_ result: PHPickerResult) async throws -> NativeHostPickedFile? {
        let provider = result.itemProvider
        let imageType = provider.registeredTypeIdentifiers
            .compactMap(UTType.init)
            .first(where: { $0.conforms(to: .image) }) ?? .image
        guard provider.hasItemConformingToTypeIdentifier(imageType.identifier) else {
            return nil
        }
        let baseName = provider.suggestedName?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty ?? "photo"
        let preferredExtension = imageType.preferredFilenameExtension ?? "jpg"
        let fileName = Self.ensuredFileName(baseName, preferredFilenameExtension: preferredExtension)
        let mimeType = imageType.preferredMIMEType ??
            UTType(filenameExtension: NSString(string: fileName).pathExtension)?.preferredMIMEType ??
            "application/octet-stream"

        return try await withCheckedThrowingContinuation { continuation in
            provider.loadDataRepresentation(forTypeIdentifier: imageType.identifier) { data, error in
                if let error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let data else {
                    continuation.resume(returning: nil)
                    return
                }
                continuation.resume(returning: NativeHostPickedFile(name: fileName, mimeType: mimeType, data: data))
            }
        }
    }

    private func loadDocumentResults(_ urls: [URL]) async throws -> [NativeHostPickedFile] {
        var files: [NativeHostPickedFile] = []
        for url in urls {
            files.append(try await loadDocumentResult(url))
        }
        return files
    }

    private func loadDocumentResult(_ url: URL) async throws -> NativeHostPickedFile {
        try await Task.detached(priority: .userInitiated) {
            let accessed = url.startAccessingSecurityScopedResource()
            defer {
                if accessed {
                    url.stopAccessingSecurityScopedResource()
                }
            }

            let data = try Data(contentsOf: url)
            let resourceValues = try? url.resourceValues(forKeys: [.contentTypeKey, .nameKey])
            let name = (resourceValues?.name ?? url.lastPathComponent)
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty ?? "attachment"
            let mimeType = resourceValues?.contentType?.preferredMIMEType ??
                UTType(filenameExtension: url.pathExtension)?.preferredMIMEType ??
                "application/octet-stream"
            return NativeHostPickedFile(name: name, mimeType: mimeType, data: data)
        }.value
    }

    private func normalizedPromptTitle(_ title: String) -> String {
        title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Add Photos & Files"
    }

    private func persistPickedFilesToTemporaryURLs(_ files: [NativeHostPickedFile]) throws -> [URL] {
        guard !files.isEmpty else {
            return []
        }

        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("codex-mobile-open-panel", isDirectory: true)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)

        return try files.enumerated().map { index, file in
            let sanitizedName = Self.ensuredFileName(
                sanitizedTemporaryFileName(file.name),
                preferredFilenameExtension: suggestedTemporaryFileExtension(for: file)
            )
            let destination = uniqueTemporaryFileURL(
                directory: directory,
                baseName: NSString(string: sanitizedName).deletingPathExtension,
                fileExtension: NSString(string: sanitizedName).pathExtension,
                index: index
            )
            try file.data.write(to: destination, options: .atomic)
            return destination
        }
    }

    private func suggestedTemporaryFileExtension(for file: NativeHostPickedFile) -> String {
        if let preferred = UTType(mimeType: file.mimeType)?.preferredFilenameExtension,
           !preferred.isEmpty {
            return preferred
        }
        let pathExtension = NSString(string: file.name).pathExtension
        if !pathExtension.isEmpty {
            return pathExtension
        }
        return "dat"
    }

    private func uniqueTemporaryFileURL(directory: URL, baseName: String, fileExtension: String, index: Int) -> URL {
        let normalizedBaseName = baseName.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "attachment"
        let normalizedExtension = fileExtension.trimmingCharacters(in: .whitespacesAndNewlines)
        var candidate = directory.appendingPathComponent(
            "\(normalizedBaseName)-\(UUID().uuidString.lowercased())-\(index)",
            isDirectory: false
        )
        if !normalizedExtension.isEmpty {
            candidate.appendPathExtension(normalizedExtension)
        }
        return candidate
    }

    private func sanitizedTemporaryFileName(_ value: String) -> String {
        let name = value.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "attachment"
        let sanitized = name.replacingOccurrences(of: "/", with: "-")
            .replacingOccurrences(of: ":", with: "-")
        return sanitized.isEmpty ? "attachment" : sanitized
    }

    private static func ensuredFileName(_ name: String, preferredFilenameExtension: String) -> String {
        if NSString(string: name).pathExtension.isEmpty == false {
            return name
        }
        let normalizedExtension = preferredFilenameExtension
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: ".", with: "")
        guard !normalizedExtension.isEmpty else {
            return name
        }
        return name + "." + normalizedExtension
    }
}

@MainActor
final class NativeHostDesktopFileBrowser {
    typealias Loader = (String?) async throws -> NativeHostDesktopDirectoryListing
    typealias RecentDirectoriesProvider = () -> [String]
    typealias RecentDirectoriesSaver = ([String]) -> Void
    typealias ThumbnailLoader = (NativeHostDesktopDirectoryEntry) async -> Data?

    private weak var presentingController: UIViewController?
    private let title: String
    private let recentDirectoriesProvider: RecentDirectoriesProvider
    private let recentDirectoriesSaver: RecentDirectoriesSaver
    private let thumbnailLoader: ThumbnailLoader
    private let loader: Loader
    private weak var navigationController: UINavigationController?
    private var continuation: CheckedContinuation<[NativeHostDesktopFileReference], Error>?

    init(
        presentingController: UIViewController,
        title: String,
        recentDirectoriesProvider: @escaping RecentDirectoriesProvider,
        recentDirectoriesSaver: @escaping RecentDirectoriesSaver,
        thumbnailLoader: @escaping ThumbnailLoader,
        loader: @escaping Loader
    ) {
        self.presentingController = presentingController
        self.title = title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Desktop Files"
        self.recentDirectoriesProvider = recentDirectoriesProvider
        self.recentDirectoriesSaver = recentDirectoriesSaver
        self.thumbnailLoader = thumbnailLoader
        self.loader = loader
    }

    func pickFiles() async throws -> [NativeHostDesktopFileReference] {
        try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
            guard let presentingController else {
                continuation.resume(throwing: NativeHostBridgeError.requestFailed("Codex Mobile could not present the desktop file browser."))
                return
            }

            let controller = NativeHostDesktopFileBrowserViewController(
                title: title,
                recentDirectoriesProvider: recentDirectoriesProvider,
                recentDirectoriesSaver: recentDirectoriesSaver,
                thumbnailLoader: thumbnailLoader,
                loader: loader
            ) { [weak self] references in
                self?.finish(with: references)
            } onCancel: { [weak self] in
                self?.finish(with: [])
            }
            let navigationController = UINavigationController(rootViewController: controller)
            navigationController.modalPresentationStyle = .formSheet
            navigationController.isModalInPresentation = true
            self.navigationController = navigationController
            presentingController.present(navigationController, animated: true)
        }
    }

    private func finish(with references: [NativeHostDesktopFileReference]) {
        let continuation = continuation
        self.continuation = nil
        navigationController?.dismiss(animated: true)
        continuation?.resume(returning: references)
    }
}

@MainActor
private final class NativeHostDesktopFileTableViewCell: UITableViewCell {
    var representedPath: String?
}

@MainActor
final class NativeHostDesktopFileBrowserViewController: UITableViewController, UISearchResultsUpdating {
    typealias Loader = (String?) async throws -> NativeHostDesktopDirectoryListing
    typealias RecentDirectoriesProvider = () -> [String]
    typealias RecentDirectoriesSaver = ([String]) -> Void
    typealias ThumbnailLoader = (NativeHostDesktopDirectoryEntry) async -> Data?

    private let browserTitle: String
    private let recentDirectoriesProvider: RecentDirectoriesProvider
    private let recentDirectoriesSaver: RecentDirectoriesSaver
    private let thumbnailLoader: ThumbnailLoader
    private let loader: Loader
    private let onComplete: ([NativeHostDesktopFileReference]) -> Void
    private let onCancel: () -> Void

    private var currentDirectoryPath: String?
    private var entries: [NativeHostDesktopDirectoryEntry] = []
    private var recentDirectories: [String] = []
    private var selectedFiles: [String: NativeHostDesktopFileReference] = [:]
    private var thumbnailImages: [String: UIImage] = [:]
    private var thumbnailFailures = Set<String>()
    private var thumbnailTasks: [String: Task<Void, Never>] = [:]
    private var isLoadingDirectory = false
    private var loadErrorMessage: String?
    private var loadTask: Task<Void, Never>?
    private lazy var searchController = UISearchController(searchResultsController: nil)

    private lazy var cancelButton = UIBarButtonItem(
        title: "Cancel",
        style: .plain,
        target: self,
        action: #selector(cancelTapped)
    )
    private lazy var addButton = UIBarButtonItem(
        title: "Add",
        style: .prominent,
        target: self,
        action: #selector(addTapped)
    )
    private lazy var upButton = UIBarButtonItem(
        title: "Up",
        style: .plain,
        target: self,
        action: #selector(upTapped)
    )

    init(
        title: String,
        recentDirectoriesProvider: @escaping RecentDirectoriesProvider,
        recentDirectoriesSaver: @escaping RecentDirectoriesSaver,
        thumbnailLoader: @escaping ThumbnailLoader,
        loader: @escaping Loader,
        onComplete: @escaping ([NativeHostDesktopFileReference]) -> Void,
        onCancel: @escaping () -> Void
    ) {
        browserTitle = title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Desktop Files"
        self.recentDirectoriesProvider = recentDirectoriesProvider
        self.recentDirectoriesSaver = recentDirectoriesSaver
        self.thumbnailLoader = thumbnailLoader
        self.loader = loader
        self.onComplete = onComplete
        self.onCancel = onCancel
        super.init(style: .insetGrouped)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    deinit {
        loadTask?.cancel()
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.title = browserTitle
        navigationItem.leftBarButtonItem = cancelButton
        searchController.obscuresBackgroundDuringPresentation = false
        searchController.searchResultsUpdater = self
        searchController.searchBar.placeholder = "Search this folder"
        navigationItem.searchController = searchController
        definesPresentationContext = true
        tableView.keyboardDismissMode = .onDrag
        recentDirectories = makeRecentDirectoriesExcludingCurrent(recentDirectoriesProvider())
        loadDirectory(nil)
        updateChrome()
    }

    override func numberOfSections(in tableView: UITableView) -> Int {
        showsRecentDirectoriesSection ? 2 : 1
    }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        if showsRecentDirectoriesSection && section == 0 {
            return recentDirectories.count
        }
        return visibleEntries.count
    }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if showsRecentDirectoriesSection && indexPath.section == 0 {
            let directoryPath = recentDirectories[indexPath.row]
            let cell = tableView.dequeueReusableCell(withIdentifier: "desktop-file-recent") ??
                UITableViewCell(style: .subtitle, reuseIdentifier: "desktop-file-recent")
            var content = cell.defaultContentConfiguration()
            content.text = displayName(forDirectoryPath: directoryPath)
            content.secondaryText = directoryPath
            content.image = UIImage(systemName: "clock.arrow.circlepath")
            content.imageProperties.tintColor = .systemOrange
            cell.contentConfiguration = content
            cell.accessoryType = .disclosureIndicator
            cell.selectionStyle = isLoadingDirectory ? .none : .default
            return cell
        }

        let entry = visibleEntries[indexPath.row]
        let cell = (tableView.dequeueReusableCell(withIdentifier: "desktop-file-entry") as? NativeHostDesktopFileTableViewCell) ??
            NativeHostDesktopFileTableViewCell(style: .subtitle, reuseIdentifier: "desktop-file-entry")
        cell.representedPath = entry.path
        var content = cell.defaultContentConfiguration()
        content.text = entry.name
        if entry.isDirectory {
            content.secondaryText = entry.path
        } else if isSearching {
            content.secondaryText = entry.path
        } else {
            content.secondaryText = nil
        }
        content.image = thumbnailImages[entry.path] ?? UIImage(systemName: entry.isDirectory ? "folder" : "doc")
        content.imageProperties.tintColor = thumbnailImages[entry.path] == nil ? (entry.isDirectory ? .systemBlue : .secondaryLabel) : nil
        content.imageProperties.maximumSize = CGSize(width: 36, height: 36)
        content.imageProperties.cornerRadius = thumbnailImages[entry.path] == nil ? 0 : 6
        cell.contentConfiguration = content
        cell.accessoryType = entry.isDirectory ? .disclosureIndicator : (selectedFiles[entry.path] == nil ? .none : .checkmark)
        cell.selectionStyle = isLoadingDirectory ? .none : .default
        loadThumbnailIfNeeded(for: entry, cell: cell)
        return cell
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        guard !isLoadingDirectory else {
            return
        }

        if showsRecentDirectoriesSection && indexPath.section == 0 {
            loadDirectory(recentDirectories[indexPath.row])
            return
        }

        let entry = visibleEntries[indexPath.row]
        if entry.isDirectory {
            loadDirectory(entry.path)
            return
        }

        if selectedFiles[entry.path] == nil {
            selectedFiles[entry.path] = NativeHostDesktopFileReference(name: entry.name, path: entry.path)
        } else {
            selectedFiles.removeValue(forKey: entry.path)
        }
        updateChrome()
        tableView.reloadRows(at: [indexPath], with: .none)
    }

    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        if showsRecentDirectoriesSection && section == 0 {
            return "Recent"
        }
        return showsRecentDirectoriesSection ? "This Folder" : nil
    }

    func updateSearchResults(for searchController: UISearchController) {
        tableView.reloadData()
        updateChrome()
    }

    @objc
    private func cancelTapped() {
        onCancel()
    }

    @objc
    private func addTapped() {
        let references = selectedFiles.values.sorted { lhs, rhs in
            lhs.path.localizedCaseInsensitiveCompare(rhs.path) == .orderedAscending
        }
        onComplete(references)
    }

    @objc
    private func upTapped() {
        guard let parentPath = parentDirectoryPath() else {
            return
        }
        loadDirectory(parentPath)
    }

    private func loadDirectory(_ directoryPath: String?) {
        loadTask?.cancel()
        cancelThumbnailTasks()
        isLoadingDirectory = true
        loadErrorMessage = nil
        entries = []
        updateChrome()
        tableView.reloadData()

        loadTask = Task { @MainActor [weak self] in
            guard let self else {
                return
            }
            do {
                let listing = try await loader(directoryPath)
                guard !Task.isCancelled else {
                    return
                }
                currentDirectoryPath = listing.directoryPath
                recordRecentDirectory(listing.directoryPath)
                entries = listing.entries
                loadErrorMessage = nil
                isLoadingDirectory = false
                updateChrome()
                tableView.reloadData()
            } catch {
                guard !Task.isCancelled else {
                    return
                }
                currentDirectoryPath = directoryPath
                entries = []
                loadErrorMessage = (error as NSError).localizedDescription
                isLoadingDirectory = false
                updateChrome()
                tableView.reloadData()
            }
        }
    }

    private func updateChrome() {
        navigationItem.prompt = currentDirectoryPath?.nilIfEmpty
        addButton.isEnabled = !selectedFiles.isEmpty && !isLoadingDirectory
        addButton.title = selectedFiles.isEmpty ? "Add" : "Add (\(selectedFiles.count))"
        navigationItem.setRightBarButtonItems([addButton, upButton], animated: false)
        upButton.isEnabled = !isLoadingDirectory && parentDirectoryPath() != nil
        tableView.backgroundView = makeBackgroundView()
    }

    private func parentDirectoryPath() -> String? {
        guard let currentDirectoryPath,
              !currentDirectoryPath.isEmpty else {
            return nil
        }
        let parentPath = NSString(string: currentDirectoryPath).deletingLastPathComponent
        guard !parentPath.isEmpty, parentPath != currentDirectoryPath else {
            return nil
        }
        return parentPath
    }

    private func makeBackgroundView() -> UIView? {
        if totalVisibleRows > 0 {
            return nil
        }

        if isLoadingDirectory {
            let stack = UIStackView()
            stack.axis = .vertical
            stack.spacing = 12
            stack.alignment = .center

            let indicator = UIActivityIndicatorView(style: .medium)
            indicator.startAnimating()
            stack.addArrangedSubview(indicator)

            let label = UILabel()
            label.text = "Loading desktop files…"
            label.textColor = .secondaryLabel
            label.font = .preferredFont(forTextStyle: .body)
            stack.addArrangedSubview(label)
            return stack
        }

        let message: String
        if let loadErrorMessage, !loadErrorMessage.isEmpty {
            message = "Unable to load this folder.\n\(loadErrorMessage)"
        } else if isSearching {
            message = "No files match \"\(searchText)\"."
        } else if entries.isEmpty {
            message = "No files in this folder."
        } else {
            return nil
        }

        let label = UILabel()
        label.text = message
        label.textColor = .secondaryLabel
        label.font = .preferredFont(forTextStyle: .body)
        label.textAlignment = .center
        label.numberOfLines = 0
        return label
    }

    private var isSearching: Bool {
        searchText.isEmpty == false
    }

    private var searchText: String {
        searchController.searchBar.text?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty ?? ""
    }

    private var showsRecentDirectoriesSection: Bool {
        isSearching == false && recentDirectories.isEmpty == false
    }

    private var visibleEntries: [NativeHostDesktopDirectoryEntry] {
        guard isSearching else {
            return entries
        }
        let normalizedSearch = searchText.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: .current)
        return entries.filter { entry in
            entry.name.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: .current).contains(normalizedSearch) ||
                entry.path.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: .current).contains(normalizedSearch)
        }
    }

    private var totalVisibleRows: Int {
        let recentCount = showsRecentDirectoriesSection ? recentDirectories.count : 0
        return recentCount + visibleEntries.count
    }

    private func recordRecentDirectory(_ directoryPath: String) {
        let normalizedPath = directoryPath.trimmingCharacters(in: .whitespacesAndNewlines)
        guard normalizedPath.isEmpty == false else {
            recentDirectories = makeRecentDirectoriesExcludingCurrent(recentDirectoriesProvider())
            return
        }

        var nextRecentDirectories = recentDirectoriesProvider().filter {
            $0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false &&
                $0 != normalizedPath
        }
        nextRecentDirectories.insert(normalizedPath, at: 0)
        recentDirectoriesSaver(nextRecentDirectories)
        recentDirectories = makeRecentDirectoriesExcludingCurrent(nextRecentDirectories)
    }

    private func makeRecentDirectoriesExcludingCurrent(_ directories: [String]) -> [String] {
        directories.filter {
            let trimmed = $0.trimmingCharacters(in: .whitespacesAndNewlines)
            guard trimmed.isEmpty == false else {
                return false
            }
            if let currentDirectoryPath {
                return trimmed != currentDirectoryPath
            }
            return true
        }
    }

    private func displayName(forDirectoryPath directoryPath: String) -> String {
        let lastPathComponent = NSString(string: directoryPath).lastPathComponent
        if lastPathComponent.isEmpty == false {
            return lastPathComponent
        }
        return directoryPath
    }

    private func cancelThumbnailTasks() {
        thumbnailTasks.values.forEach { $0.cancel() }
        thumbnailTasks.removeAll()
    }

    private func loadThumbnailIfNeeded(for entry: NativeHostDesktopDirectoryEntry, cell: NativeHostDesktopFileTableViewCell) {
        guard thumbnailEligible(entry),
              thumbnailImages[entry.path] == nil,
              thumbnailFailures.contains(entry.path) == false,
              thumbnailTasks[entry.path] == nil else {
            return
        }

        let entryPath = entry.path
        thumbnailTasks[entryPath] = Task { @MainActor [weak self, weak cell] in
            guard let self else {
                return
            }
            defer {
                thumbnailTasks.removeValue(forKey: entryPath)
            }

            guard let data = await thumbnailLoader(entry),
                  Task.isCancelled == false,
                  let image = UIImage(data: data) else {
                thumbnailFailures.insert(entryPath)
                return
            }

            thumbnailImages[entryPath] = image
            if let cell, cell.representedPath == entryPath,
               let indexPath = tableView.indexPath(for: cell) {
                tableView.reloadRows(at: [indexPath], with: .none)
            }
        }
    }

    private func thumbnailEligible(_ entry: NativeHostDesktopDirectoryEntry) -> Bool {
        guard entry.isDirectory == false else {
            return false
        }
        let fileExtension = URL(fileURLWithPath: entry.path).pathExtension.trimmingCharacters(in: .whitespacesAndNewlines)
        guard fileExtension.isEmpty == false,
              let type = UTType(filenameExtension: fileExtension) else {
            return false
        }
        return type.conforms(to: .image)
    }
}

@MainActor
final class NativeHostDesktopThumbnailCache {
    private enum CachedValue {
        case hit(Data)
        case miss
    }

    private let cache = NSCache<NSString, NSData>()
    private var cachedValues: [String: CachedValue] = [:]
    private var inflightTasks: [String: Task<Data?, Never>] = [:]

    init() {
        cache.countLimit = 96
        cache.totalCostLimit = 12 * 1024 * 1024
    }

    func data(for key: String, producer: @escaping () async -> Data?) async -> Data? {
        if let cachedData = cache.object(forKey: key as NSString) {
            return cachedData as Data
        }
        if let cachedValue = cachedValues[key] {
            switch cachedValue {
            case let .hit(data):
                return data
            case .miss:
                return nil
            }
        }
        if let task = inflightTasks[key] {
            return await task.value
        }

        let task = Task<Data?, Never> {
            await producer()
        }
        inflightTasks[key] = task
        let result = await task.value
        inflightTasks.removeValue(forKey: key)

        if let result {
            cache.setObject(result as NSData, forKey: key as NSString, cost: result.count)
            cachedValues[key] = .hit(result)
        } else {
            cachedValues[key] = .miss
        }
        return result
    }

    func reset() {
        inflightTasks.values.forEach { $0.cancel() }
        inflightTasks.removeAll()
        cachedValues.removeAll()
        cache.removeAllObjects()
    }
}
