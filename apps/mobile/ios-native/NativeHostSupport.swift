import Foundation
import SwiftUI

typealias JSONDictionary = [String: Any]

enum NativeHostDebugLog {
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

enum NativeHostBridgeError: LocalizedError {
    case invalidPayload(String)
    case invalidURL(String)
    case invalidResponse(String)
    case requestFailed(String)
    case webSocketClosed(String)

    var errorDescription: String? {
        switch self {
        case let .invalidPayload(message),
             let .invalidURL(message),
             let .invalidResponse(message),
             let .requestFailed(message),
             let .webSocketClosed(message):
            return message
        }
    }
}

enum ThemeMode: String, CaseIterable, Identifiable {
    case system
    case light
    case dark

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .system:
            return "Follow System"
        case .light:
            return "Light"
        case .dark:
            return "Dark"
        }
    }

    var preferredColorScheme: ColorScheme? {
        switch self {
        case .system:
            return nil
        case .light:
            return .light
        case .dark:
            return .dark
        }
    }
}

final class NativeHostPreferencesStore {
    private let defaults = UserDefaults.standard
    private let themeModeKey = "codex.nativeHost.preferences.themeMode"
    private let autoResumeKey = "codex.nativeHost.preferences.autoResumeActiveSession"
    private let fullDesktopFileAccessKey = "codex.nativeHost.preferences.fullDesktopFileAccess"
    private let recentDesktopDirectoriesKey = "codex.nativeHost.preferences.recentDesktopDirectories"
    private let maxRecentDesktopDirectories = 8

    func readThemeMode() -> ThemeMode {
        ThemeMode(rawValue: defaults.string(forKey: themeModeKey) ?? "") ?? .system
    }

    func writeThemeMode(_ themeMode: ThemeMode) {
        defaults.set(themeMode.rawValue, forKey: themeModeKey)
    }

    func shouldAutoResumeActiveSession() -> Bool {
        defaults.object(forKey: autoResumeKey) as? Bool ?? true
    }

    func setAutoResumeActiveSession(_ enabled: Bool) {
        defaults.set(enabled, forKey: autoResumeKey)
    }

    func isFullDesktopFileAccessEnabled() -> Bool {
        defaults.object(forKey: fullDesktopFileAccessKey) as? Bool ?? false
    }

    func setFullDesktopFileAccessEnabled(_ enabled: Bool) {
        defaults.set(enabled, forKey: fullDesktopFileAccessKey)
    }

    func readRecentDesktopDirectories() -> [String] {
        normalizeRecentDesktopDirectories(defaults.stringArray(forKey: recentDesktopDirectoriesKey) ?? [])
    }

    func writeRecentDesktopDirectories(_ paths: [String]) {
        defaults.set(normalizeRecentDesktopDirectories(paths), forKey: recentDesktopDirectoriesKey)
    }

    private func normalizeRecentDesktopDirectories(_ paths: [String]) -> [String] {
        var seen = Set<String>()
        var result: [String] = []
        for path in paths {
            let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty, seen.contains(trimmed) == false else {
                continue
            }
            seen.insert(trimmed)
            result.append(trimmed)
            if result.count == maxRecentDesktopDirectories {
                break
            }
        }
        return result
    }
}

struct BridgeProfile: Equatable, Identifiable {
    let id: String
    let bridgeID: String?
    let name: String
    let serverEndpoint: String
    let authToken: String?
    let lastUsedAtMilliseconds: Int64?
}

struct ConnectionTargetResponse: Equatable {
    let bridgeID: String?
    let recommendedServerEndpoint: String
    let authMode: String
    let localAuthPage: String?
}

struct PairingResponse: Equatable {
    let accessToken: String
    let approved: Bool
}

struct BridgeLoadTarget: Equatable {
    let baseURL: URL
    let usesLocalProxy: Bool
}

struct HttpProxyResponse {
    let body: Any?
    let status: Int
    let headers: JSONDictionary
}

struct BridgeBootstrapState {
    let persistedAtomState: JSONDictionary?
    let workspaceRootOptions: [String]
    let activeWorkspaceRoots: [String]
    let workspaceRootLabels: [String: String]
    let pinnedThreadIDs: [String]
    let globalState: JSONDictionary
}

struct NativeHostPickedFile {
    let name: String
    let mimeType: String
    let data: Data
}

struct NativeHostDesktopFileReference: Equatable, Identifiable {
    let name: String
    let path: String

    var id: String { path }
}

struct NativeHostDesktopDirectoryEntry: Equatable, Identifiable {
    let name: String
    let path: String
    let isDirectory: Bool

    var id: String { path }
}

struct NativeHostDesktopDirectoryListing: Equatable {
    let directoryPath: String
    let entries: [NativeHostDesktopDirectoryEntry]
}

enum NativeHostBrowserSessionSource: String, CaseIterable, Identifiable {
    case preview
    case attach

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .preview:
            return "Preview Browser"
        case .attach:
            return "Attach Chrome Tab"
        }
    }

    var subtitle: String {
        switch self {
        case .preview:
            return "Launch a dedicated browser session for previews."
        case .attach:
            return "Control a tab from your real Chrome session."
        }
    }
}

enum NativeHostBrowserViewportPreset: String, CaseIterable, Identifiable {
    case desktop
    case tablet
    case mobile

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .desktop:
            return "Desktop"
        case .tablet:
            return "Tablet"
        case .mobile:
            return "Mobile"
        }
    }
}

struct NativeHostBrowserAttachTarget: Equatable, Identifiable {
    let id: String
    let title: String
    let url: String
    let debugBaseURL: String
}

struct NativeHostBrowserSessionSummary: Equatable {
    let sessionID: String
    let source: NativeHostBrowserSessionSource
    let title: String
    let url: String
    let loading: Bool
    let isStreaming: Bool
    let textInputActive: Bool
    let editableText: String
    let selectionStart: Int
    let selectionEnd: Int
    let canGoBack: Bool
    let canGoForward: Bool
    let width: Int
    let height: Int
    let scale: Double
    let debugBaseURL: String?
    let targetID: String?
}

struct NativeHostBrowserSessionSnapshot: Equatable {
    let summary: NativeHostBrowserSessionSummary
    let revision: Int
    let mimeType: String
    let imageData: Data
    let consoleErrorCount: Int
    let consoleWarnCount: Int
    let consoleLines: [String]
    let networkInflightCount: Int
    let networkFailedCount: Int
    let networkFailedLines: [String]
}

struct NativeHostDesktopSessionSummary: Equatable {
    let sessionID: String
    let title: String
    let isStreaming: Bool
    let width: Int
    let height: Int
    let scale: Double
    let preferredTransport: String
    let videoCodec: String?
    let videoReady: Bool
    let videoError: String?
    let lastError: String?
    let textInputActive: Bool
    let editableText: String
    let editablePlaceholder: String
    let editableRole: String?
    let selectionStart: Int
    let selectionEnd: Int
}

struct NativeHostDesktopWebRTCOffer: Equatable {
    let peerID: String
    let sdp: String
}

actor BrowserSessionFrameStreamClient {
    private let webSocketURL: URL
    private let headers: [String: String]
    private let onFrame: @Sendable (Data) async -> Void
    private let onStatus: @Sendable (JSONDictionary) async -> Void
    private let onDisconnect: @Sendable (Error?) -> Void

    private var session: URLSession?
    private var socket: URLSessionWebSocketTask?
    private var receiveLoopTask: Task<Void, Never>?
    private var pingLoopTask: Task<Void, Never>?
    private var disconnectNotified = false
    private var closeRequested = false

    init(
        webSocketURL: URL,
        headers: [String: String],
        onFrame: @escaping @Sendable (Data) async -> Void,
        onStatus: @escaping @Sendable (JSONDictionary) async -> Void,
        onDisconnect: @escaping @Sendable (Error?) -> Void
    ) {
        self.webSocketURL = webSocketURL
        self.headers = headers
        self.onFrame = onFrame
        self.onStatus = onStatus
        self.onDisconnect = onDisconnect
    }

    func connect() {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 20
        configuration.timeoutIntervalForResource = 20
        let session = URLSession(configuration: configuration)
        var request = URLRequest(url: webSocketURL)
        headers.forEach { request.setValue($1, forHTTPHeaderField: $0) }
        let socket = session.webSocketTask(with: request)

        closeRequested = false
        disconnectNotified = false
        self.session = session
        self.socket = socket
        socket.resume()

        receiveLoopTask = Task { [weak self] in
            guard let self else { return }
            await self.receiveLoop(using: socket)
        }
        pingLoopTask = Task { [weak self] in
            guard let self else { return }
            await self.runPingLoop(using: socket)
        }
    }

    func close() async {
        closeRequested = true
        disconnectNotified = true
        pingLoopTask?.cancel()
        receiveLoopTask?.cancel()
        socket?.cancel(with: .normalClosure, reason: nil)
        session?.invalidateAndCancel()
        socket = nil
        session = nil
        pingLoopTask = nil
        receiveLoopTask = nil
    }

    private func receiveLoop(using socket: URLSessionWebSocketTask) async {
        do {
            while !Task.isCancelled {
                let message = try await socket.receive()
                switch message {
                case let .data(data):
                    await onFrame(data)
                case let .string(text):
                    guard let payload = parseJSONValue(from: text) as? JSONDictionary,
                          let type = (payload["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                          type == "status",
                          let statusPayload = payload["payload"] as? JSONDictionary else {
                        continue
                    }
                    await onStatus(deepCopyJSONObject(statusPayload))
                @unknown default:
                    continue
                }
            }
        } catch {
            await handleDisconnect(error)
        }
    }

    private func runPingLoop(using socket: URLSessionWebSocketTask) async {
        while !Task.isCancelled {
            do {
                try await Task.sleep(nanoseconds: 15_000_000_000)
                try Task.checkCancellation()
                try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
                    socket.sendPing { error in
                        if let error {
                            continuation.resume(throwing: error)
                        } else {
                            continuation.resume(returning: ())
                        }
                    }
                }
            } catch is CancellationError {
                return
            } catch {
                await handleDisconnect(error)
                return
            }
        }
    }

    private func handleDisconnect(_ error: Error?) async {
        if disconnectNotified || closeRequested {
            return
        }
        disconnectNotified = true
        onDisconnect(error)
    }
}

enum NativeHostAttachmentSelection {
    case deviceFiles([NativeHostPickedFile])
    case desktopFiles([NativeHostDesktopFileReference])
}

enum NativeHostConnectionStage: String, CaseIterable {
    case payloadReceived
    case pairingDevice
    case openingWorkspace

    var title: String {
        switch self {
        case .payloadReceived:
            return "Code Received"
        case .pairingDevice:
            return "Approving This iPad"
        case .openingWorkspace:
            return "Opening Workspace"
        }
    }
}

enum ShellChromeState {
    case disconnected
    case loading
    case connected
    case error
}

struct WorkspaceSelectionChange {
    let optionsChanged: Bool
    let activeChanged: Bool
}

final class NativeHostSessionState {
    private(set) var workspaceRootOptions: [String] = []
    private(set) var activeWorkspaceRoots: [String] = []
    private(set) var workspaceRootLabels: [String: String] = [:]
    private(set) var pinnedThreadIDs: [String] = []

    func updateWorkspaceRoots(_ nextRoots: [String], preferredRoot: String? = nil) -> WorkspaceSelectionChange? {
        let normalizedRoots = uniqueTrimmedStrings(nextRoots.compactMap(normalizeWorkspaceRootCandidate))
        let preferred = normalizeWorkspaceRootCandidate(preferredRoot)
        let nextActiveRoots: [String]
        if let preferred, normalizedRoots.contains(preferred) {
            nextActiveRoots = [preferred]
        } else if let active = activeWorkspaceRoots.first, normalizedRoots.contains(active) {
            nextActiveRoots = [active]
        } else if let first = normalizedRoots.first {
            nextActiveRoots = [first]
        } else {
            nextActiveRoots = []
        }

        let optionsChanged = workspaceRootOptions != normalizedRoots
        let activeChanged = activeWorkspaceRoots != nextActiveRoots
        guard optionsChanged || activeChanged else {
            return nil
        }

        workspaceRootOptions = normalizedRoots
        activeWorkspaceRoots = nextActiveRoots
        return WorkspaceSelectionChange(optionsChanged: optionsChanged, activeChanged: activeChanged)
    }

    func mergeWorkspaceRoots(_ nextRoots: [String], preferredRoot: String? = nil) -> WorkspaceSelectionChange? {
        updateWorkspaceRoots(workspaceRootOptions + nextRoots, preferredRoot: preferredRoot)
    }

    func setActiveWorkspaceRoot(_ root: String) -> WorkspaceSelectionChange? {
        guard let normalizedRoot = normalizeWorkspaceRootCandidate(root) else {
            return nil
        }

        let nextOptions = workspaceRootOptions.contains(normalizedRoot)
            ? workspaceRootOptions
            : [normalizedRoot] + workspaceRootOptions
        let nextActiveRoots = [normalizedRoot]
        let optionsChanged = workspaceRootOptions != nextOptions
        let activeChanged = activeWorkspaceRoots != nextActiveRoots
        guard optionsChanged || activeChanged else {
            return nil
        }

        workspaceRootOptions = nextOptions
        activeWorkspaceRoots = nextActiveRoots
        return WorkspaceSelectionChange(optionsChanged: optionsChanged, activeChanged: activeChanged)
    }

    func replaceWorkspaceRootLabels(_ labels: [String: String]) {
        workspaceRootLabels = labels
    }

    func renameWorkspaceRoot(_ root: String, label: String) -> Bool {
        guard let normalizedRoot = normalizeWorkspaceRootCandidate(root), workspaceRootOptions.contains(normalizedRoot) else {
            return false
        }
        if label.isEmpty {
            workspaceRootLabels.removeValue(forKey: normalizedRoot)
        } else {
            workspaceRootLabels[normalizedRoot] = label
        }
        return true
    }

    func setThreadPinned(threadID: String, pinned: Bool) -> Bool {
        let normalizedThreadID = threadID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedThreadID.isEmpty else {
            return false
        }
        if pinned {
            pinnedThreadIDs = uniqueTrimmedStrings(pinnedThreadIDs + [normalizedThreadID])
        } else {
            pinnedThreadIDs.removeAll { $0 == normalizedThreadID }
        }
        return true
    }

    func replacePinnedThreadIDs(_ threadIDs: [String]) {
        pinnedThreadIDs = uniqueTrimmedStrings(threadIDs)
    }
}

func jsonStringArray(_ value: Any?) -> [String] {
    guard let values = value as? [Any] else {
        return []
    }
    return values.compactMap { item in
        guard let text = item as? String else {
            return nil
        }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

func jsonObjectStringMap(_ value: Any?) -> [String: String] {
    guard let values = value as? JSONDictionary else {
        return [:]
    }
    return values.reduce(into: [String: String]()) { partialResult, entry in
        guard let text = entry.value as? String else {
            return
        }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }
        partialResult[entry.key] = trimmed
    }
}

func uniqueTrimmedStrings<S: Sequence>(_ values: S) -> [String] where S.Element == String {
    var seen = Set<String>()
    var result: [String] = []
    for value in values {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, seen.insert(trimmed).inserted else {
            continue
        }
        result.append(trimmed)
    }
    return result
}

func deepCopyJSONValue(_ value: Any?) -> Any? {
    switch value {
    case nil:
        return nil
    case is NSNull:
        return NSNull()
    case let dictionary as JSONDictionary:
        return dictionary.reduce(into: JSONDictionary()) { partialResult, entry in
            partialResult[entry.key] = deepCopyJSONValue(entry.value) ?? NSNull()
        }
    case let array as [Any]:
        return array.map { deepCopyJSONValue($0) ?? NSNull() }
    case let string as String:
        return string
    case let number as NSNumber:
        return number
    case let boolean as Bool:
        return boolean
    case let integer as Int:
        return integer
    case let integer as Int64:
        return integer
    case let doubleValue as Double:
        return doubleValue
    case let floatValue as Float:
        return Double(floatValue)
    default:
        return String(describing: value)
    }
}

func deepCopyJSONObject(_ value: JSONDictionary) -> JSONDictionary {
    deepCopyJSONValue(value) as? JSONDictionary ?? [:]
}

func toJSONCompatible(_ value: Any?) -> Any {
    deepCopyJSONValue(value) ?? NSNull()
}

func jsonEncodedString(_ value: Any) -> String {
    guard JSONSerialization.isValidJSONObject(value),
          let data = try? JSONSerialization.data(withJSONObject: value, options: []),
          let text = String(data: data, encoding: .utf8) else {
        return "null"
    }
    return text
}

func jsonFragmentString(_ value: Any?) -> String {
    let wrapped = [toJSONCompatible(value)]
    guard JSONSerialization.isValidJSONObject(wrapped),
          let data = try? JSONSerialization.data(withJSONObject: wrapped, options: []),
          var text = String(data: data, encoding: .utf8) else {
        return "null"
    }
    if text.hasPrefix("[") {
        text.removeFirst()
    }
    if text.hasSuffix("]") {
        text.removeLast()
    }
    return text
}

func jsonObject(from data: Data) throws -> JSONDictionary {
    let object = try JSONSerialization.jsonObject(with: data, options: [])
    guard let dictionary = object as? JSONDictionary else {
        throw NativeHostBridgeError.invalidResponse("Expected a JSON object.")
    }
    return dictionary
}

func jsonObject(from value: Any?) -> JSONDictionary? {
    value as? JSONDictionary
}

func parseJSONValue(from text: String) -> Any? {
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else {
        return nil
    }
    if let data = trimmed.data(using: .utf8),
       let object = try? JSONSerialization.jsonObject(with: data, options: []) {
        return object
    }
    return trimmed
}

func normalizeWorkspaceRootCandidate(_ value: String?) -> String? {
    let normalized = value?
        .trimmingCharacters(in: .whitespacesAndNewlines)
        .replacingOccurrences(of: "\\", with: "/")
        .replacingOccurrences(of: "/+", with: "/", options: .regularExpression) ?? ""
    return normalized.isEmpty ? nil : normalized
}

func compactLabel(_ value: String, maxLength: Int = 28) -> String {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.count > maxLength else {
        return trimmed
    }
    return String(trimmed.prefix(maxLength - 1)).trimmingCharacters(in: .whitespacesAndNewlines) + "…"
}

func displayProfileLabel(_ profile: BridgeProfile) -> String {
    let normalizedEndpoint = BridgeAPI.normalizeEndpoint(profile.serverEndpoint)
    let rawName = profile.name.trimmingCharacters(in: .whitespacesAndNewlines)
    if !rawName.isEmpty, rawName != normalizedEndpoint, !rawName.hasPrefix("http") {
        let candidate: String
        if rawName.contains("."), !rawName.contains(" ") {
            candidate = String(rawName.split(separator: ".").first ?? Substring(rawName))
        } else {
            candidate = rawName
        }
        return compactLabel(candidate)
    }

    guard let host = URL(string: BridgeAPI.deriveServerHTTPBaseURL(normalizedEndpoint))?.host, !host.isEmpty else {
        return compactLabel(
            normalizedEndpoint
                .replacingOccurrences(of: "https://", with: "")
                .replacingOccurrences(of: "http://", with: "")
                .replacingOccurrences(of: "ws://", with: "")
                .replacingOccurrences(of: "wss://", with: "")
        )
    }
    let candidate = host.split(separator: ".").first.map(String.init) ?? host
    return compactLabel(candidate)
}

func displayProfileDetail(_ profile: BridgeProfile) -> String {
    let normalizedEndpoint = BridgeAPI.normalizeEndpoint(profile.serverEndpoint)
    if let url = URL(string: BridgeAPI.deriveServerHTTPBaseURL(normalizedEndpoint)),
       let host = url.host,
       !host.isEmpty {
        let portSuffix = url.port.map { ":\($0)" } ?? ""
        let remainder = host.split(separator: ".").dropFirst().joined(separator: ".")
        let candidate = remainder.isEmpty ? host : remainder
        return compactLabel(candidate + portSuffix, maxLength: 28)
    }
    return compactLabel(
        normalizedEndpoint
            .replacingOccurrences(of: "https://", with: "")
            .replacingOccurrences(of: "http://", with: "")
            .replacingOccurrences(of: "ws://", with: "")
            .replacingOccurrences(of: "wss://", with: ""),
        maxLength: 28
    )
}

func profileAvatar(_ profile: BridgeProfile) -> String {
    let tokens = displayProfileLabel(profile)
        .split(whereSeparator: { "-_ .".contains($0) })
        .map(String.init)
        .filter { !$0.isEmpty }
    let preferredToken = tokens.last ?? displayProfileLabel(profile)
    return preferredToken.first.map { String($0).uppercased() } ?? "C"
}

func describeProfileStatus(_ profile: BridgeProfile, isCurrent: Bool) -> String {
    guard let lastUsedAt = profile.lastUsedAtMilliseconds else {
        return isCurrent ? "Ready on this iPad" : "Saved on this iPad"
    }
    let formatter = RelativeDateTimeFormatter()
    formatter.unitsStyle = .short
    let date = Date(timeIntervalSince1970: TimeInterval(lastUsedAt) / 1000)
    return "Used \(formatter.localizedString(for: date, relativeTo: Date()))"
}

extension String {
    var nilIfEmpty: String? {
        isEmpty ? nil : self
    }
}

extension Optional where Wrapped == String {
    var nilIfEmpty: String? {
        switch self {
        case let .some(value) where !value.isEmpty:
            return value
        default:
            return nil
        }
    }
}
