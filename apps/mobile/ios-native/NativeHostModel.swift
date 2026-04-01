import Foundation
import SwiftUI
import UIKit

private enum NativeHostDebugLog {
    private static let fileName = "codex-mobile-debug.log"

    private static var fileURL: URL? {
        FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first?.appendingPathComponent(fileName)
    }

    static func reset() {
        guard let fileURL else {
            return
        }
        try? FileManager.default.removeItem(at: fileURL)
        write("session-start")
    }

    static func write(_ message: String) {
        guard let fileURL,
              let data = formattedLine(message).data(using: .utf8) else {
            return
        }

        if FileManager.default.fileExists(atPath: fileURL.path) == false {
            FileManager.default.createFile(atPath: fileURL.path, contents: data)
            return
        }

        do {
            let handle = try FileHandle(forWritingTo: fileURL)
            defer { try? handle.close() }
            try handle.seekToEnd()
            try handle.write(contentsOf: data)
        } catch {
            try? data.write(to: fileURL, options: .atomic)
        }
    }

    private static func formattedLine(_ message: String) -> String {
        let timestamp = ISO8601DateFormatter().string(from: Date())
        return "\(timestamp) \(message)\n"
    }
}

@MainActor
final class NativeHostModel: ObservableObject {
    @Published private(set) var profiles: [BridgeProfile] = []
    @Published var activeProfile: BridgeProfile?
    @Published private(set) var chromeState: ShellChromeState = .disconnected
    @Published private(set) var statusMessage = "Choose how to connect"
    @Published private(set) var currentConnectionStage: NativeHostConnectionStage?
    @Published private(set) var currentConnectionRequiresPairing = false
    @Published var themeMode: ThemeMode
    @Published var sessionsSheetPresented = false
    @Published var settingsSheetPresented = false
    @Published var scannerPresented = false
    @Published var systemColorScheme: ColorScheme = .dark
    @Published var profilePendingReset: BridgeProfile?

    let webBridge = NativeHostWebBridge()

    private let profileStore: BridgeProfileStore
    private let preferencesStore: NativeHostPreferencesStore
    private let runtime: NativeHostBridgeRuntime

    private var activeLoadTarget: BridgeLoadTarget?
    private var bridgeLoadGeneration = 0
    private var autoReconnectAttemptedProfileID: String?
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
    var hasSavedProfiles: Bool { !profiles.isEmpty }
    var savedProfilesPreview: [BridgeProfile] { Array(profiles.prefix(2)) }
    var activeSessionForSheet: BridgeProfile? {
        if let activeProfile, !isTransientProfile(activeProfile) {
            return activeProfile
        }
        return profileStore.readActive()
    }
    var shouldShowWorkspace: Bool { activeProfile != nil && chromeState == .connected }
    var shouldKeepWebViewMounted: Bool { activeProfile != nil && (chromeState == .connected || chromeState == .loading) }

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

    func handleScannedPayload(_ rawPayload: String) {
        NSLog("CodexMobile scanned QR payload length=%d", rawPayload.count)
        NativeHostDebugLog.write("scanned-payload length=\(rawPayload.count)")
        scannerPresented = false
        importEnrollment(rawPayload)
    }

    func activateProfile(_ profile: BridgeProfile) {
        profileStore.setActive(profile.id)
        refreshProfiles()
        openBridge(profile)
    }

    func reloadActiveBridge() {
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
        if normalized.localizedCaseInsensitiveContains("expired") {
            return "QR Code Expired"
        }
        return "Couldn’t open workspace"
    }

    func presentSessionsSheet() {
        sessionsSheetPresented = true
    }

    func presentSettingsSheet() {
        settingsSheetPresented = true
    }

    func importEnrollment(_ rawJSON: String) {
        let trimmed = rawJSON.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            statusMessage = "Scan a desktop QR code first."
            return
        }

        scannerPresented = false

        do {
            switch try EnrollmentParser.parse(rawJSON: trimmed) {
            case let .bridge(bridgeID, name, serverEndpoint, pairingCode):
                NSLog("CodexMobile importEnrollment type=bridge endpoint=%@", serverEndpoint)
                NativeHostDebugLog.write("import type=bridge bridgeID=\(bridgeID ?? "<nil>") endpoint=\(serverEndpoint) pairing=\(pairingCode != nil)")
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
                        existingAuthToken: nil
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
        existingAuthToken: String?
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
                lastUsedAtMilliseconds: nil
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
        activeProfile = profile
        activeLoadTarget = nil
        refreshProfiles()

        if currentConnectionStage == nil {
            updateConnectionProgress(
                bridgeID: profile.bridgeID,
                profileName: profile.name,
                endpoint: profile.serverEndpoint,
                stage: .openingWorkspace,
                requiresPairing: false
            )
        }

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

    private func scheduleAutoReconnectIfNeeded(for profile: BridgeProfile) {
        guard preferencesStore.shouldAutoResumeActiveSession(),
              chromeState == .error,
              autoReconnectAttemptedProfileID != profile.id else {
            return
        }

        autoReconnectAttemptedProfileID = profile.id
        runtimeSetStatus("Trying to restore this session")
        autoReconnectTask?.cancel()
        autoReconnectTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 1_200_000_000)
            guard let self,
                  self.activeProfile?.id == profile.id,
                  self.chromeState == .error else {
                return
            }
            self.reloadActiveBridge()
        }
    }

    private func resetAutoReconnectState() {
        autoReconnectTask?.cancel()
        autoReconnectTask = nil
        autoReconnectAttemptedProfileID = nil
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

    func runtimeOpenExternalURL(_ url: URL) {
        UIApplication.shared.open(url)
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
}
