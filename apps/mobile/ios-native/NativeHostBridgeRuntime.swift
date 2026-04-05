import Foundation

@MainActor
protocol NativeHostBridgeRuntimeDelegate: AnyObject {
    var runtimeActiveProfile: BridgeProfile? { get }
    var runtimeActiveLoadTarget: BridgeLoadTarget? { get }
    func runtimeResolveBridgeLoadTarget(for profile: BridgeProfile) async throws -> BridgeLoadTarget
    func runtimeSetStatus(_ message: String)
    func runtimeNormalizeError(_ error: Error) -> String
    func runtimeHandleBridgeConnectionIssue(_ error: Error)
    func runtimeOpenExternalURL(_ url: URL)
    func runtimePickFiles(title: String) async throws -> NativeHostAttachmentSelection
}

@MainActor
final class NativeHostBridgeRuntime {
    private static let localHostID = "local"
    private static let localHostConfig: JSONDictionary = [
        "id": localHostID,
        "hostId": localHostID,
        "display_name": "Local",
        "displayName": "Local",
        "kind": "local",
    ]
    private static let defaultPersistedAtomState: JSONDictionary = [
        "statsig_default_enable_features": [
            "fast_mode": true,
        ],
    ]

    weak var delegate: NativeHostBridgeRuntimeDelegate?

    private var javaScriptEvaluator: ((String) -> Void)?
    private var persistedAtomState: JSONDictionary
    private var globalState: JSONDictionary = [:]
    private var configurationState: JSONDictionary = [:]
    private var sharedObjects: JSONDictionary
    private var sharedObjectSubscribers: [String: Int] = [:]
    private let sessionState = NativeHostSessionState()
    private let appServerCoordinator: NativeHostAppServerCoordinator

    init() {
        persistedAtomState = deepCopyJSONObject(Self.defaultPersistedAtomState)
        sharedObjects = [
            "pending_worktrees": [Any](),
            "remote_connections": [Any](),
            "host_config": deepCopyJSONObject(Self.localHostConfig),
        ]
        appServerCoordinator = NativeHostAppServerCoordinator()
        appServerCoordinator.runtime = self
    }

    func setJavaScriptEvaluator(_ evaluator: @escaping (String) -> Void) {
        javaScriptEvaluator = evaluator
    }

    func reset() {
        Task {
            await appServerCoordinator.reset()
        }
    }

    func handleBridgeConnectionIssue(_ error: Error) {
        delegate?.runtimeHandleBridgeConnectionIssue(error)
    }

    func syncSavedConnectionsState(profiles: [BridgeProfile], activeProfileID: String?) {
        sharedObjects["remote_connections"] = profiles.map { profile in
            [
                "id": profile.id,
                "name": profile.name,
                "serverEndpoint": profile.serverEndpoint,
                "active": profile.id == activeProfileID,
                "tailnetManaged": false,
            ] as JSONDictionary
        }
        if (sharedObjectSubscribers["remote_connections"] ?? 0) > 0 {
            broadcastSharedObjectUpdate(key: "remote_connections", value: sharedObjects["remote_connections"])
        }
    }

    func hydrateBridgeBootstrapState(loadTarget: BridgeLoadTarget, authToken: String?) async throws -> BridgeBootstrapState {
        let snapshot = try await BridgeAPI.fetchGlobalStateSnapshot(
            baseURL: loadTarget.baseURL.absoluteString,
            authToken: authToken
        )
        let state = snapshot["state"] as? JSONDictionary ?? [:]
        let workspacePayload = snapshot["workspaceRootOptions"] as? JSONDictionary

        let persistedState = state["electron-persisted-atom-state"] as? JSONDictionary
        let workspaceRoots = {
            let directRoots = jsonStringArray(workspacePayload?["roots"])
            if !directRoots.isEmpty {
                return directRoots
            }
            return jsonStringArray(state["project-order"])
                + jsonStringArray(state["electron-saved-workspace-roots"])
                + jsonStringArray(state["active-workspace-roots"])
        }()
        let activeRoots = {
            let directRoots = jsonStringArray(workspacePayload?["activeRoots"])
            if !directRoots.isEmpty {
                return directRoots
            }
            return jsonStringArray(state["active-workspace-roots"])
        }()
        let workspaceLabels = {
            let directLabels = jsonObjectStringMap(workspacePayload?["labels"])
            if !directLabels.isEmpty {
                return directLabels
            }
            return jsonObjectStringMap(state["workspace-root-labels"])
        }()
        let pinnedThreadIDs = jsonStringArray(state["pinned-thread-ids"])

        var globalSnapshot: JSONDictionary = [:]
        state.forEach { key, value in
            globalSnapshot[key] = deepCopyJSONValue(value) ?? NSNull()
        }

        return BridgeBootstrapState(
            persistedAtomState: persistedState.map(deepCopyJSONObject),
            workspaceRootOptions: uniqueTrimmedStrings(workspaceRoots),
            activeWorkspaceRoots: uniqueTrimmedStrings(activeRoots),
            workspaceRootLabels: workspaceLabels,
            pinnedThreadIDs: uniqueTrimmedStrings(pinnedThreadIDs),
            globalState: globalSnapshot
        )
    }

    func applyBridgeBootstrapState(_ state: BridgeBootstrapState) {
        persistedAtomState = deepCopyJSONObject(Self.defaultPersistedAtomState)
        state.persistedAtomState?.forEach { key, value in
            persistedAtomState[key] = toJSONCompatible(value)
        }

        globalState.removeAll()
        state.globalState.forEach { key, value in
            globalState[key] = deepCopyJSONValue(value) ?? NSNull()
        }

        sessionState.replaceWorkspaceRootLabels(state.workspaceRootLabels)
        sessionState.replacePinnedThreadIDs(state.pinnedThreadIDs)
        updateWorkspaceRoots(state.workspaceRootOptions, preferredRoot: state.activeWorkspaceRoots.first)
    }

    func handleEnvelope(_ rawMessage: String) {
        guard let envelope = parseJSONValue(from: rawMessage) as? JSONDictionary,
              envelope["__codexMobile"] as? Bool == true,
              let kind = envelope["kind"] as? String else {
            return
        }

        switch kind {
        case "preload-ready":
            NativeHostDebugLog.write("preload-ready")
            if let profile = delegate?.runtimeActiveProfile {
                delegate?.runtimeSetStatus("Connected to \(profile.serverEndpoint)")
            }
        case "console":
            guard let payload = envelope["payload"] as? JSONDictionary else {
                return
            }
            NativeHostDebugLog.write("preload-console \(summarizeConsolePayload(payload))")
        case "bridge-send-message":
            guard let payload = envelope["payload"] as? JSONDictionary else {
                return
            }
            handleRendererMessage(payload)
        case "bridge-send-worker-message":
            guard let payload = envelope["payload"] as? JSONDictionary else {
                return
            }
            handleWorkerBridgeMessage(payload)
        case "bridge-show-context-menu", "bridge-show-application-menu":
            guard let payload = envelope["payload"] as? JSONDictionary,
                  let requestID = (payload["requestId"] as? String)?
                    .trimmingCharacters(in: .whitespacesAndNewlines),
                  !requestID.isEmpty else {
                return
            }
            sendBridgeResponse(requestID: requestID, result: nil)
        case "bridge-host-call":
            guard let payload = envelope["payload"] as? JSONDictionary,
                  let requestID = (payload["requestId"] as? String)?
                    .trimmingCharacters(in: .whitespacesAndNewlines),
                  !requestID.isEmpty else {
                return
            }
            handleBridgeHostCall(requestID: requestID, payload: payload)
        default:
            break
        }
    }

    private func handleRendererReady() {
        sendPersistedAtomSync()
        sendHostMessage([
            "type": "custom-prompts-updated",
            "prompts": [],
        ])
        sendHostMessage([
            "type": "app-update-ready-changed",
            "isUpdateReady": false,
        ])
        sendHostMessage([
            "type": "electron-window-focus-changed",
            "isFocused": true,
        ])
        sendHostMessage(["type": "workspace-root-options-updated"])
        sendHostMessage(["type": "active-workspace-roots-updated"])
        sharedObjects.forEach { key, value in
            broadcastSharedObjectUpdate(key: key, value: value)
        }
    }

    private func handleRendererMessage(_ message: JSONDictionary) {
        guard let type = message["type"] as? String else {
            return
        }

        switch type {
        case "ready":
            handleRendererReady()
        case "persisted-atom-sync-request":
            sendPersistedAtomSync()
        case "persisted-atom-update":
            let key = (message["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return
            }
            let deleted = message["deleted"] as? Bool ?? false
            if deleted {
                persistedAtomState.removeValue(forKey: key)
            } else {
                persistedAtomState[key] = toJSONCompatible(message["value"])
            }
            sendHostMessage([
                "type": "persisted-atom-updated",
                "key": key,
                "value": deleted ? NSNull() : toJSONCompatible(message["value"]),
                "deleted": deleted,
            ])
        case "persisted-atom-reset":
            persistedAtomState = deepCopyJSONObject(Self.defaultPersistedAtomState)
            sendPersistedAtomSync()
        case "shared-object-subscribe":
            let key = (message["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return
            }
            sharedObjectSubscribers[key] = (sharedObjectSubscribers[key] ?? 0) + 1
            broadcastSharedObjectUpdate(key: key, value: sharedObjects[key])
        case "shared-object-unsubscribe":
            let key = (message["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return
            }
            let count = sharedObjectSubscribers[key] ?? 0
            if count <= 1 {
                sharedObjectSubscribers.removeValue(forKey: key)
            } else {
                sharedObjectSubscribers[key] = count - 1
            }
        case "shared-object-set":
            let key = (message["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return
            }
            sharedObjects[key] = deepCopyJSONValue(message["value"]) ?? NSNull()
            broadcastSharedObjectUpdate(key: key, value: sharedObjects[key])
        case "electron-window-focus-request":
            sendHostMessage([
                "type": "electron-window-focus-changed",
                "isFocused": true,
            ])
        case "open-in-browser":
            guard let urlString = (message["url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                  let url = URL(string: urlString) else {
                return
            }
            delegate?.runtimeOpenExternalURL(url)
        case "terminal-create", "terminal-attach":
            let sessionID = (message["sessionId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !sessionID.isEmpty else {
                return
            }
            sendHostMessage([
                "type": "terminal-attached",
                "sessionId": sessionID,
                "cwd": message["cwd"] as? String ?? "",
                "shell": "zsh",
            ])
            sendHostMessage([
                "type": "terminal-init-log",
                "sessionId": sessionID,
                "log": "",
            ])
        case "terminal-close":
            let sessionID = (message["sessionId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !sessionID.isEmpty else {
                return
            }
            sendHostMessage([
                "type": "terminal-exit",
                "sessionId": sessionID,
                "code": 0,
                "signal": NSNull(),
            ])
        case "workspace-root-option-picked", "electron-set-active-workspace-root":
            let root = (message["root"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !root.isEmpty else {
                return
            }
            setActiveWorkspaceRoot(root)
        case "electron-update-workspace-root-options":
            updateWorkspaceRoots(jsonStringArray(message["roots"]))
        case "electron-rename-workspace-root-option":
            let root = message["root"] as? String ?? ""
            let label = (message["label"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if sessionState.renameWorkspaceRoot(root, label: label) {
                sendHostMessage(["type": "workspace-root-options-updated"])
            }
        case "fetch":
            Task { [weak self] in
                await self?.handleFetchMessage(message)
            }
        case "fetch-stream":
            let requestID = (message["requestId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !requestID.isEmpty else {
                return
            }
            sendHostMessage([
                "type": "fetch-stream-error",
                "requestId": requestID,
                "error": "Streaming fetch is not supported in Codex Mobile yet.",
            ])
        case "mcp-request":
            if let request = message["request"] as? JSONDictionary,
               let method = request["method"] as? String,
               method == "pick-files" {
                NativeHostDebugLog.write("renderer-mcp-request method=pick-files")
            }
            Task { [weak self] in
                await self?.handleMCPRequestMessage(message)
            }
        case "cancel-fetch", "cancel-fetch-stream", "bridge-unimplemented", "view-focused",
             "power-save-blocker-set", "electron-set-window-mode", "electron-request-microphone-permission",
             "electron-set-badge-count", "desktop-notification-hide", "desktop-notification-show",
             "install-app-update", "open-debug-window", "open-thread-overlay", "thread-stream-state-changed",
             "set-telemetry-user", "toggle-trace-recording", "hotkey-window-enabled-changed",
             "electron-desktop-features-changed", "log-message":
            break
        default:
            break
        }
    }

    private func handleWorkerBridgeMessage(_ payload: JSONDictionary) {
        let workerID = (payload["workerId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !workerID.isEmpty,
              let workerPayload = payload["payload"] as? JSONDictionary,
              workerPayload["type"] as? String == "worker-request",
              let request = workerPayload["request"] as? JSONDictionary,
              let requestID = (request["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
              let method = (request["method"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
              !requestID.isEmpty,
              !method.isEmpty else {
            return
        }

        let result: JSONDictionary
        if workerID == "git", method == "stable-metadata" {
            result = [
                "type": "ok",
                "value": [
                    "cwd": "",
                    "root": "",
                    "commonDir": "",
                    "gitDir": NSNull(),
                    "branch": NSNull(),
                    "upstreamBranch": NSNull(),
                    "headSha": NSNull(),
                    "originUrl": NSNull(),
                    "isRepository": false,
                    "isWorktree": false,
                    "worktreeRoot": "",
                ],
            ]
        } else {
            result = [
                "type": "error",
                "error": [
                    "message": "Unsupported worker request: \(workerID)/\(method)",
                ],
            ]
        }

        sendWorkerMessage(
            workerID: workerID,
            payload: [
                "type": "worker-response",
                "workerId": workerID,
                "response": [
                    "id": requestID,
                    "method": method,
                    "result": result,
                ],
            ]
        )
    }

    private func handleFetchMessage(_ message: JSONDictionary) async {
        let requestID = (message["requestId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let rawURL = (message["url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !requestID.isEmpty, !rawURL.isEmpty,
              let delegate,
              let profile = delegate.runtimeActiveProfile else {
            return
        }

        do {
            let loadTarget = try await resolveActiveLoadTarget(profile: profile, delegate: delegate)
            if let method = resolveRequestMethodName(rawURL) {
                let body = parseJSONValue(from: message["body"] as? String ?? "") as? JSONDictionary
                if method == "pick-files" {
                    NativeHostDebugLog.write("fetch-pick-files start")
                    do {
                        let files = try await pickAndUploadFiles(
                            title: resolveAttachmentPickerTitle(from: body ?? [:]),
                            delegate: delegate,
                            loadTarget: loadTarget,
                            authToken: resolveRequestAuthToken(loadTarget: loadTarget, profile: profile)
                        )
                        NativeHostDebugLog.write("fetch-pick-files success count=\(files.count)")
                        sendHostMessage(buildFetchSuccessResponse(
                            requestID: requestID,
                            body: [
                                "files": files,
                            ],
                            status: 200,
                            headers: [:]
                        ))
                    } catch {
                        let message = delegate.runtimeNormalizeError(error)
                        NativeHostDebugLog.write("fetch-pick-files failed message=\(message)")
                        sendHostMessage(buildFetchErrorResponse(
                            requestID: requestID,
                            error: message,
                            status: 500
                        ))
                    }
                    return
                }
                if let localResult = resolveFetchMethodPayload(method: method, params: body) {
                    sendHostMessage(buildFetchSuccessResponse(
                        requestID: requestID,
                        body: localResult,
                        status: 200,
                        headers: [:]
                    ))
                    return
                }

                let result = try await BridgeAPI.performDirectRPC(
                    baseURL: loadTarget.baseURL.absoluteString,
                    method: method,
                    params: body ?? [:],
                    authToken: resolveRequestAuthToken(loadTarget: loadTarget, profile: profile)
                )
                integrateDirectRPCResult(method: method, result: result)
                sendHostMessage(buildFetchSuccessResponse(
                    requestID: requestID,
                    body: result,
                    status: 200,
                    headers: [:]
                ))
                return
            }

            guard let proxyURL = resolveServerFetchURL(rawURL: rawURL, baseURL: loadTarget.baseURL) else {
                sendHostMessage(buildFetchErrorResponse(
                    requestID: requestID,
                    error: "Unsupported fetch URL: \(rawURL)",
                    status: 501
                ))
                return
            }

            let response = try await BridgeAPI.proxyHTTPFetch(
                url: proxyURL,
                method: message["method"] as? String,
                requestBody: (message["body"] as? String)?.nilIfEmpty,
                requestHeaders: message["headers"] as? JSONDictionary,
                authToken: resolveRequestAuthToken(loadTarget: loadTarget, profile: profile)
            )
            sendHostMessage(buildFetchSuccessResponse(
                requestID: requestID,
                body: response.body,
                status: response.status,
                headers: response.headers
            ))
        } catch {
            sendHostMessage(buildFetchErrorResponse(
                requestID: requestID,
                error: delegate.runtimeNormalizeError(error),
                status: 500
            ))
            if resolveRequestMethodName(rawURL) != nil {
                delegate.runtimeHandleBridgeConnectionIssue(error)
            }
        }
    }

    private func handleMCPRequestMessage(_ message: JSONDictionary) async {
        guard let request = message["request"] as? JSONDictionary,
              let requestID = request["id"],
              let method = (request["method"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
              !method.isEmpty,
              let delegate,
              let profile = delegate.runtimeActiveProfile else {
            return
        }

        do {
            let loadTarget = try await resolveActiveLoadTarget(profile: profile, delegate: delegate)
            let requestParams = request["params"] as? JSONDictionary ?? [:]
            let authToken = resolveRequestAuthToken(loadTarget: loadTarget, profile: profile)

            if method == "pick-files" {
                NativeHostDebugLog.write("runtime-pick-files start")
                let result: JSONDictionary = [
                    "files": try await pickAndUploadFiles(
                        title: resolveAttachmentPickerTitle(from: requestParams),
                        delegate: delegate,
                        loadTarget: loadTarget,
                        authToken: authToken
                    ),
                ]
                let fileCount = (result["files"] as? [Any])?.count ?? 0
                NativeHostDebugLog.write("runtime-pick-files success count=\(fileCount)")
                sendHostMessage([
                    "type": "mcp-response",
                    "hostId": Self.localHostID,
                    "id": requestID,
                    "result": result,
                    "message": [
                        "id": requestID,
                        "result": result,
                    ],
                    "response": [
                        "id": requestID,
                        "result": result,
                    ],
                ])
                return
            }

            if let localResult = resolveFetchMethodPayload(method: method, params: requestParams) as? JSONDictionary {
                sendHostMessage([
                    "type": "mcp-response",
                    "hostId": Self.localHostID,
                    "id": requestID,
                    "result": localResult,
                    "message": [
                        "id": requestID,
                        "result": localResult,
                    ],
                    "response": [
                        "id": requestID,
                        "result": localResult,
                    ],
                ])
                return
            }

            let result: JSONDictionary
            do {
                result = try await appServerCoordinator.performRequest(
                    loadTarget: loadTarget,
                    authToken: authToken,
                    method: method,
                    params: requestParams
                )
            } catch {
                result = try await BridgeAPI.performDirectRPC(
                    baseURL: loadTarget.baseURL.absoluteString,
                    method: method,
                    params: requestParams,
                    authToken: authToken
                )
            }

            integrateDirectRPCResult(method: method, result: result)
            sendHostMessage([
                "type": "mcp-response",
                "hostId": Self.localHostID,
                "id": requestID,
                "result": result,
                "message": [
                    "id": requestID,
                    "result": result,
                ],
                "response": [
                    "id": requestID,
                    "result": result,
                ],
            ])

            if method == "turn/start" {
                let threadID = (requestParams["threadId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                let turnID = ((result["turn"] as? JSONDictionary)?["id"] as? String)?
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                    .nilIfEmpty
                appServerCoordinator.rememberPendingTurnCompletion(threadID: threadID, turnID: turnID)
                if let threadID {
                    appServerCoordinator.scheduleTurnCompletionFallback(
                        threadID: threadID,
                        turnID: turnID,
                        loadTarget: loadTarget,
                        authToken: authToken
                    )
                }
            } else {
                appServerCoordinator.reconcileThreadSnapshotIfNeeded(
                    method: method,
                    result: result,
                    loadTarget: loadTarget,
                    authToken: authToken
                )
            }
        } catch {
            if method == "pick-files" {
                NativeHostDebugLog.write("runtime-pick-files failed message=\(delegate.runtimeNormalizeError(error))")
            }
            let normalizedError = delegate.runtimeNormalizeError(error)
            sendHostMessage([
                "type": "mcp-response",
                "hostId": Self.localHostID,
                "id": requestID,
                "error": [
                    "message": normalizedError,
                ],
                "message": [
                    "id": requestID,
                    "error": [
                        "message": normalizedError,
                    ],
                ],
                "response": [
                    "id": requestID,
                    "error": [
                        "message": normalizedError,
                    ],
                ],
            ])
            delegate.runtimeHandleBridgeConnectionIssue(error)
        }
    }

    private func pickAndUploadFiles(
        title: String,
        delegate: NativeHostBridgeRuntimeDelegate,
        loadTarget: BridgeLoadTarget,
        authToken: String?
    ) async throws -> [JSONDictionary] {
        switch try await delegate.runtimePickFiles(title: title) {
        case let .deviceFiles(pickedFiles):
            NativeHostDebugLog.write("pick-and-upload selected source=device count=\(pickedFiles.count)")
            return try await uploadPickedFiles(
                pickedFiles,
                loadTarget: loadTarget,
                authToken: authToken
            )
        case let .desktopFiles(references):
            NativeHostDebugLog.write("pick-and-upload selected source=desktop count=\(references.count)")
            return references.map { reference in
                [
                    "label": reference.name,
                    "path": reference.path,
                    "fsPath": reference.path,
                ]
            }
        }
    }

    private func resolveAttachmentPickerTitle(from params: JSONDictionary) -> String {
        if let pickerTitle = (params["pickerTitle"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines),
           !pickerTitle.isEmpty {
            return pickerTitle
        }
        if let title = (params["title"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines),
           !title.isEmpty {
            return title
        }
        return ""
    }

    private func uploadPickedFiles(
        _ files: [NativeHostPickedFile],
        loadTarget: BridgeLoadTarget,
        authToken: String?
    ) async throws -> [JSONDictionary] {
        if files.isEmpty {
            return []
        }

        var uploadedFiles: [JSONDictionary] = []
        for file in files {
            NativeHostDebugLog.write("upload-picked-file start name=\(file.name) size=\(file.data.count)")
            let result = try await BridgeAPI.performDirectRPC(
                baseURL: loadTarget.baseURL.absoluteString,
                method: "mobile/upload-picked-file",
                params: [
                    "name": file.name,
                    "mimeType": file.mimeType,
                    "contentsBase64": file.data.base64EncodedString(),
                ],
                authToken: authToken
            )
            if let errorMessage = (result["error"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines),
               !errorMessage.isEmpty {
                throw NativeHostBridgeError.requestFailed(errorMessage)
            }

            let label = (result["label"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty ?? file.name
            let path = (result["path"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty
            let fsPath = (result["fsPath"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty
            guard let path, let fsPath else {
                throw NativeHostBridgeError.invalidResponse("Picked file upload did not return a desktop path.")
            }

            uploadedFiles.append([
                "label": label,
                "path": path,
                "fsPath": fsPath,
            ])
            NativeHostDebugLog.write("upload-picked-file success path=\(fsPath)")
        }
        return uploadedFiles
    }

    private func summarizeConsolePayload(_ payload: JSONDictionary) -> String {
        let tag = (payload["tag"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "unknown"
        let trace = (payload["trace"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "console"
        let rawPayload = payload["payload"]
        let serializedPayload: String
        if let rawPayload {
            serializedPayload = jsonFragmentString(rawPayload)
        } else {
            serializedPayload = "null"
        }
        return "trace=\(trace) tag=\(tag) payload=\(serializedPayload)"
    }

    private func resolveFetchMethodPayload(method: String, params: JSONDictionary?) -> Any? {
        switch method {
        case "get-global-state":
            let key = (params?["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            return [
                "value": toJSONCompatible(key.flatMap { globalState[$0] }),
            ]
        case "set-global-state":
            let key = (params?["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return ["success": false]
            }
            if params?.keys.contains("value") == true {
                globalState[key] = deepCopyJSONValue(params?["value"]) ?? NSNull()
            } else {
                globalState.removeValue(forKey: key)
            }
            return ["success": true]
        case "get-configuration":
            let key = (params?["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            return [
                "value": toJSONCompatible(key.flatMap { configurationState[$0] }),
            ]
        case "set-configuration":
            let key = (params?["key"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !key.isEmpty else {
                return ["success": false]
            }
            if params?.keys.contains("value") == true {
                configurationState[key] = deepCopyJSONValue(params?["value"]) ?? NSNull()
            } else {
                configurationState.removeValue(forKey: key)
            }
            return ["success": true]
        case "list-pinned-threads":
            return ["threadIds": sessionState.pinnedThreadIDs]
        case "set-thread-pinned":
            let threadID = (params?["threadId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !threadID.isEmpty else {
                return [
                    "success": false,
                    "threadIds": sessionState.pinnedThreadIDs,
                ]
            }
            _ = sessionState.setThreadPinned(threadID: threadID, pinned: params?["pinned"] as? Bool ?? false)
            return [
                "success": true,
                "threadIds": sessionState.pinnedThreadIDs,
            ]
        case "set-pinned-threads-order":
            let nextThreadIDs = uniqueTrimmedStrings(
                ((params?["threadIds"] as? [Any]) ?? [])
                    .compactMap { $0 as? String }
            )
            sessionState.replacePinnedThreadIDs(nextThreadIDs)
            return [
                "success": true,
                "threadIds": sessionState.pinnedThreadIDs,
            ]
        case "experimentalFeature/list":
            return [
                "data": [Any](),
                "nextCursor": NSNull(),
            ]
        case "os-info":
            return [
                "platform": "ios",
                "isMacOS": false,
                "isWindows": false,
                "isLinux": false,
            ]
        case "locale-info":
            return deriveLocaleInfo()
        case "active-workspace-roots":
            return ["roots": sessionState.activeWorkspaceRoots]
        case "workspace-root-options":
            return [
                "roots": sessionState.workspaceRootOptions,
                "activeRoots": sessionState.activeWorkspaceRoots,
                "labels": sessionState.workspaceRootLabels,
            ]
        case "paths-exist":
            let paths = ((params?["paths"] as? [Any]) ?? [])
                .compactMap { $0 as? String }
                .compactMap(normalizeWorkspaceRootCandidate)
            return ["existingPaths": paths]
        default:
            return nil
        }
    }

    fileprivate func integrateDirectRPCResult(method: String, result: JSONDictionary) {
        switch method {
        case "thread/list", "thread/read", "thread/start", "thread/resume":
            mergeWorkspaceRootsFromResult(result)
        case "workspace-root-options":
            sessionState.replaceWorkspaceRootLabels(jsonObjectStringMap(result["labels"]))
            updateWorkspaceRoots(
                jsonStringArray(result["roots"]),
                preferredRoot: jsonStringArray(result["activeRoots"]).first
            )
        default:
            break
        }
    }

    fileprivate func emitMCPNotification(method: String, params: JSONDictionary) {
        sendHostMessage([
            "type": "mcp-notification",
            "hostId": Self.localHostID,
            "method": method,
            "params": params,
            "notification": [
                "method": method,
                "params": params,
            ],
        ])
    }

    fileprivate func emitSyntheticTurnCompletion(threadID: String, turn: JSONDictionary, emittedTurnCompletions: inout [String: String]) {
        let turnID = (turn["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !turnID.isEmpty, emittedTurnCompletions[threadID] != turnID else {
            return
        }

        emitMCPNotification(method: "turn/started", params: [
            "threadId": threadID,
            "turn": turn,
        ])

        if let items = turn["items"] as? [Any] {
            for item in items.compactMap({ $0 as? JSONDictionary }) where (item["type"] as? String) != "userMessage" {
                let itemPayload: JSONDictionary = [
                    "threadId": threadID,
                    "turnId": turnID,
                    "item": item,
                ]
                emitMCPNotification(method: "item/started", params: itemPayload)
                emitMCPNotification(method: "item/completed", params: itemPayload)
            }
        }

        emitMCPNotification(method: "turn/completed", params: [
            "threadId": threadID,
            "turn": turn,
        ])
        emittedTurnCompletions[threadID] = turnID
    }

    fileprivate func sendHostMessage(_ message: JSONDictionary) {
        injectHostMessages([message])
    }

    fileprivate func sendBridgeResponse(requestID: String, result: Any?) {
        let script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (host && typeof host.resolveBridgeRequest === "function") {
                host.resolveBridgeRequest(\(jsonEncodedString(requestID)), \(jsonFragmentString(result)));
              }
            })();
            """
        javaScriptEvaluator?(script)
    }

    private func handleBridgeHostCall(requestID: String, payload: JSONDictionary) {
        Task { @MainActor [weak self] in
            guard let self else { return }
            await self.handleBridgeHostCallAsync(requestID: requestID, payload: payload)
        }
    }

    private func handleBridgeHostCallAsync(requestID: String, payload: JSONDictionary) async {
        let method = normalizeBridgeHostMethodName(payload["method"] as? String)
        guard let delegate,
              let profile = delegate.runtimeActiveProfile else {
            sendBridgeResponse(requestID: requestID, result: [
                "supported": false,
                "error": "No active desktop session is available.",
            ])
            return
        }

        let params = payload["params"] as? JSONDictionary ?? [:]
        switch method {
        case "attachment/pick":
            do {
                let loadTarget = try await resolveActiveLoadTarget(profile: profile, delegate: delegate)
                let authToken = resolveRequestAuthToken(loadTarget: loadTarget, profile: profile)
                let files = try await pickAndUploadFiles(
                    title: resolveAttachmentPickerTitle(from: params),
                    delegate: delegate,
                    loadTarget: loadTarget,
                    authToken: authToken
                )
                sendBridgeResponse(requestID: requestID, result: [
                    "supported": true,
                    "files": files,
                ])
            } catch {
                sendBridgeResponse(requestID: requestID, result: [
                    "supported": false,
                    "error": delegate.runtimeNormalizeError(error),
                ])
                delegate.runtimeHandleBridgeConnectionIssue(error)
            }
        default:
            sendBridgeResponse(requestID: requestID, result: [
                "supported": false,
                "error": "Unsupported Codex host method: \(method.isEmpty ? "unknown" : method)",
            ])
        }
    }

    private func normalizeBridgeHostMethodName(_ value: String?) -> String {
        (value ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: ".", with: "/")
    }

    fileprivate func sendWorkerMessage(workerID: String, payload: JSONDictionary) {
        let script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (host && typeof host.dispatchWorkerMessage === "function") {
                host.dispatchWorkerMessage(\(jsonEncodedString(workerID)), \(jsonEncodedString(payload)));
              }
            })();
            """
        javaScriptEvaluator?(script)
    }

    private func sendPersistedAtomSync() {
        sendHostMessage([
            "type": "persisted-atom-sync",
            "state": deepCopyJSONObject(persistedAtomState),
        ])
    }

    private func broadcastSharedObjectUpdate(key: String, value: Any?) {
        sendHostMessage([
            "type": "shared-object-updated",
            "key": key,
            "value": toJSONCompatible(value),
        ])
    }

    private func updateWorkspaceRoots(_ nextRoots: [String], preferredRoot: String? = nil) {
        emitWorkspaceSelectionChange(
            sessionState.updateWorkspaceRoots(nextRoots, preferredRoot: preferredRoot)
        )
    }

    private func mergeWorkspaceRoots(_ nextRoots: [String], preferredRoot: String? = nil) {
        emitWorkspaceSelectionChange(
            sessionState.mergeWorkspaceRoots(nextRoots, preferredRoot: preferredRoot)
        )
    }

    private func setActiveWorkspaceRoot(_ root: String) {
        emitWorkspaceSelectionChange(sessionState.setActiveWorkspaceRoot(root))
    }

    private func emitWorkspaceSelectionChange(_ change: WorkspaceSelectionChange?) {
        guard let change else {
            return
        }
        if change.optionsChanged {
            sendHostMessage(["type": "workspace-root-options-updated"])
        }
        if change.activeChanged {
            sendHostMessage(["type": "active-workspace-roots-updated"])
        }
    }

    private func mergeWorkspaceRootsFromResult(_ result: JSONDictionary) {
        var candidates: [String] = []

        func collectThreadRoots(_ value: JSONDictionary?) {
            if let cwd = normalizeWorkspaceRootCandidate(value?["cwd"] as? String) {
                candidates.append(cwd)
            }
        }

        if let cwd = normalizeWorkspaceRootCandidate(result["cwd"] as? String) {
            candidates.append(cwd)
        }
        collectThreadRoots(result["thread"] as? JSONDictionary)

        if let data = result["data"] as? [Any] {
            for item in data.compactMap({ $0 as? JSONDictionary }) {
                collectThreadRoots(item)
            }
        }

        if !candidates.isEmpty {
            mergeWorkspaceRoots(candidates, preferredRoot: candidates.first)
        }
    }

    private func deriveLocaleInfo() -> JSONDictionary {
        let locale = Locale.current.identifier.replacingOccurrences(of: "_", with: "-")
        return [
            "ideLocale": locale.isEmpty ? "en-US" : locale,
            "systemLocale": locale.isEmpty ? "en-US" : locale,
        ]
    }

    private func injectHostMessages(_ messages: [Any]) {
        guard !messages.isEmpty else {
            return
        }
        let script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (!host || typeof host.dispatchHostMessage !== "function") {
                return;
              }
              var messages = \(jsonEncodedString(messages));
              for (var index = 0; index < messages.length; index += 1) {
                host.dispatchHostMessage(messages[index]);
              }
            })();
            """
        javaScriptEvaluator?(script)
    }

    private func resolveActiveLoadTarget(profile: BridgeProfile, delegate: NativeHostBridgeRuntimeDelegate) async throws -> BridgeLoadTarget {
        if let activeLoadTarget = delegate.runtimeActiveLoadTarget {
            return activeLoadTarget
        }
        return try await delegate.runtimeResolveBridgeLoadTarget(for: profile)
    }

    private func resolveRequestAuthToken(loadTarget: BridgeLoadTarget, profile: BridgeProfile) -> String? {
        loadTarget.usesLocalProxy ? nil : profile.authToken
    }

    private func resolveRequestMethodName(_ rawURL: String) -> String? {
        let prefix = "vscode://codex/"
        guard rawURL.hasPrefix(prefix) else {
            return nil
        }
        let path = rawURL.replacingOccurrences(of: prefix, with: "")
        let method = path.split(separator: "?").first.map(String.init)?.trimmingCharacters(in: .whitespacesAndNewlines)
        return method?.isEmpty == false ? method : nil
    }

    private func resolveServerFetchURL(rawURL: String, baseURL: URL) -> URL? {
        let trimmed = rawURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }
        if let absoluteURL = URL(string: trimmed), absoluteURL.scheme != nil {
            return absoluteURL
        }
        return URL(string: trimmed, relativeTo: baseURL)?.absoluteURL
    }

    private func buildFetchSuccessResponse(requestID: String, body: Any?, status: Int, headers: JSONDictionary) -> JSONDictionary {
        [
            "type": "fetch-response",
            "requestId": requestID,
            "responseType": "success",
            "status": status,
            "headers": headers,
            "bodyJsonString": jsonFragmentString(body),
        ]
    }

    private func buildFetchErrorResponse(requestID: String, error: String, status: Int) -> JSONDictionary {
        [
            "type": "fetch-response",
            "requestId": requestID,
            "responseType": "error",
            "status": status,
            "error": error,
        ]
    }
}

@MainActor
final class NativeHostAppServerCoordinator {
    weak var runtime: NativeHostBridgeRuntime?

    private var pendingTurnCompletions: [String: String] = [:]
    private var activeTurnReconciliations = Set<String>()
    private var emittedTurnCompletions: [String: String] = [:]
    private var appServerWebSocketClient: AppServerWebSocketClient?
    private var activeWebSocketURL: URL?
    private var activeHeaders: [String: String] = [:]

    func reset() async {
        if let appServerWebSocketClient {
            await appServerWebSocketClient.close()
        }
        appServerWebSocketClient = nil
        activeWebSocketURL = nil
        activeHeaders = [:]
        pendingTurnCompletions.removeAll()
        activeTurnReconciliations.removeAll()
    }

    func performRequest(loadTarget: BridgeLoadTarget, authToken: String?, method: String, params: JSONDictionary) async throws -> JSONDictionary {
        let client = try await webSocketClient(loadTarget: loadTarget, authToken: authToken)
        return try await client.request(method: method, params: params)
    }

    func rememberPendingTurnCompletion(threadID: String?, turnID: String?) {
        guard let threadID, let turnID, !threadID.isEmpty, !turnID.isEmpty else {
            return
        }
        pendingTurnCompletions[threadID] = turnID
    }

    func scheduleTurnCompletionFallback(threadID: String, turnID: String?, loadTarget: BridgeLoadTarget, authToken: String?, delayMilliseconds: UInt64 = 1_500) {
        Task { [weak self] in
            try? await Task.sleep(nanoseconds: delayMilliseconds * 1_000_000)
            guard let self, self.hasPendingTurnCompletion(threadID: threadID, turnID: turnID) else {
                return
            }
            self.reconcileCompletedThreadState(threadID: threadID, loadTarget: loadTarget, authToken: authToken)
        }
    }

    func reconcileThreadSnapshotIfNeeded(method: String, result: JSONDictionary, loadTarget: BridgeLoadTarget, authToken: String?) {
        guard method == "thread/read" || method == "thread/resume" || method == "thread/start",
              let threadObject = result["thread"] as? JSONDictionary,
              let threadID = (threadObject["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
              !threadID.isEmpty else {
            return
        }
        let statusObject = threadObject["status"] as? JSONDictionary
        guard method == "thread/start" || isThreadStatusTerminal(statusObject) else {
            return
        }
        reconcileCompletedThreadState(threadID: threadID, loadTarget: loadTarget, authToken: authToken)
    }

    private func webSocketClient(loadTarget: BridgeLoadTarget, authToken: String?) async throws -> AppServerWebSocketClient {
        let webSocketURL = urlForWebSocket(baseURL: loadTarget.baseURL)
        let headers = requestHeaders(loadTarget: loadTarget, authToken: authToken)

        if let appServerWebSocketClient,
           activeWebSocketURL == webSocketURL,
           activeHeaders == headers {
            return appServerWebSocketClient
        }

        if let appServerWebSocketClient {
            await appServerWebSocketClient.close()
        }

        let client = AppServerWebSocketClient(
            webSocketURL: webSocketURL,
            headers: headers,
            onNotification: { [weak self] method, params in
                guard let self else { return }
                await self.handleNotification(method: method, params: params, loadTarget: loadTarget, authToken: authToken)
            },
            onLog: { message, error in
                if let error {
                    NSLog("%@: %@", message, error.localizedDescription)
                } else {
                    NSLog("%@", message)
                }
            },
            onDisconnect: { [weak self] error in
                guard let self else { return }
                Task { @MainActor [weak self] in
                    guard let self else { return }
                    self.appServerWebSocketClient = nil
                    self.activeWebSocketURL = nil
                    self.activeHeaders = [:]
                    let probe = await BridgeAPI.probeBridgeReady(
                        baseURL: loadTarget.baseURL.absoluteString,
                        authToken: authToken
                    )
                    if !probe.ready {
                        self.runtime?.handleBridgeConnectionIssue(
                            error ?? NativeHostBridgeError.webSocketClosed("App server websocket closed.")
                        )
                    }
                }
            }
        )
        appServerWebSocketClient = client
        activeWebSocketURL = webSocketURL
        activeHeaders = headers
        return client
    }

    private func handleNotification(method: String, params: JSONDictionary, loadTarget: BridgeLoadTarget, authToken: String?) {
        switch method {
        case "thread/status/changed":
            let threadID = (params["threadId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let status = params["status"] as? JSONDictionary
            if !threadID.isEmpty, isThreadStatusTerminal(status), hasPendingTurnCompletion(threadID: threadID, turnID: nil) {
                reconcileCompletedThreadState(threadID: threadID, loadTarget: loadTarget, authToken: authToken)
            }
        case "turn/started":
            let threadID = (params["threadId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            let turnID = ((params["turn"] as? JSONDictionary)?["id"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty
            rememberPendingTurnCompletion(threadID: threadID, turnID: turnID)
        case "turn/completed":
            let threadID = (params["threadId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            let turnID = ((params["turn"] as? JSONDictionary)?["id"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty
            clearPendingTurnCompletion(threadID: threadID, turnID: turnID)
        default:
            break
        }
        runtime?.emitMCPNotification(method: method, params: deepCopyJSONObject(params))
    }

    private func reconcileCompletedThreadState(threadID: String, loadTarget: BridgeLoadTarget, authToken: String?) {
        guard activeTurnReconciliations.insert(threadID).inserted else {
            return
        }

        Task { [weak self] in
            defer {
                Task { @MainActor [weak self] in
                    self?.activeTurnReconciliations.remove(threadID)
                }
            }

            for attempt in 0..<90 {
                do {
                    let result = try await BridgeAPI.performDirectRPC(
                        baseURL: loadTarget.baseURL.absoluteString,
                        method: "thread/read",
                        params: [
                            "threadId": threadID,
                            "includeTurns": true,
                        ],
                        authToken: authToken
                    )
                    await MainActor.run {
                        self?.runtime?.integrateDirectRPCResult(method: "thread/read", result: result)
                    }

                    if let threadObject = result["thread"] as? JSONDictionary,
                       let statusObject = threadObject["status"] as? JSONDictionary,
                       self?.isThreadStatusTerminal(statusObject) == true {
                        if let turns = threadObject["turns"] as? [Any],
                           let latestTurn = turns.compactMap({ $0 as? JSONDictionary }).last,
                           let latestTurnID = (latestTurn["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                           !latestTurnID.isEmpty,
                           let latestTurnStatus = (latestTurn["status"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                           !latestTurnStatus.isEmpty,
                           latestTurnStatus != "inProgress" {
                            await MainActor.run {
                                guard let self else { return }
                                self.runtime?.emitSyntheticTurnCompletion(
                                    threadID: threadID,
                                    turn: latestTurn,
                                    emittedTurnCompletions: &self.emittedTurnCompletions
                                )
                                self.clearPendingTurnCompletion(threadID: threadID, turnID: latestTurnID)
                            }
                        }
                        await MainActor.run {
                            self?.runtime?.emitMCPNotification(method: "thread/status/changed", params: [
                                "threadId": threadID,
                                "status": statusObject,
                            ])
                        }
                        return
                    }
                    if attempt < 89 {
                        try await Task.sleep(nanoseconds: attempt < 5 ? 350_000_000 : 800_000_000)
                    }
                } catch {
                    return
                }
            }
        }
    }

    private func clearPendingTurnCompletion(threadID: String?, turnID: String?) {
        guard let threadID,
              let pendingTurnID = pendingTurnCompletions[threadID] else {
            return
        }
        if let turnID, pendingTurnID != turnID {
            return
        }
        pendingTurnCompletions.removeValue(forKey: threadID)
    }

    private func hasPendingTurnCompletion(threadID: String, turnID: String?) -> Bool {
        guard let pendingTurnID = pendingTurnCompletions[threadID] else {
            return false
        }
        if let turnID, pendingTurnID != turnID {
            return false
        }
        return true
    }

    private func isThreadStatusTerminal(_ status: JSONDictionary?) -> Bool {
        let type = (status?["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return type == "idle" || type == "systemError"
    }

    private func urlForWebSocket(baseURL: URL) -> URL {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) ?? URLComponents()
        components.scheme = components.scheme == "https" ? "wss" : "ws"
        return components.url ?? baseURL
    }

    private func requestHeaders(loadTarget: BridgeLoadTarget, authToken: String?) -> [String: String] {
        guard !loadTarget.usesLocalProxy, let authToken, !authToken.isEmpty else {
            return [:]
        }
        return ["Authorization": "Bearer \(authToken)"]
    }
}
