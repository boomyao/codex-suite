import Foundation

actor AppServerWebSocketClient {
    private let webSocketURL: URL
    private let headers: [String: String]
    private let onNotification: @Sendable (String, JSONDictionary) async -> Void
    private let onLog: @Sendable (String, Error?) -> Void

    private var nextRequestID = 2
    private var pendingRequests: [Int: CheckedContinuation<JSONDictionary, Error>] = [:]
    private var session: URLSession?
    private var socket: URLSessionWebSocketTask?
    private var readyTask: Task<Void, Error>?
    private var receiveLoopTask: Task<Void, Never>?

    init(
        webSocketURL: URL,
        headers: [String: String],
        onNotification: @escaping @Sendable (String, JSONDictionary) async -> Void,
        onLog: @escaping @Sendable (String, Error?) -> Void
    ) {
        self.webSocketURL = webSocketURL
        self.headers = headers
        self.onNotification = onNotification
        self.onLog = onLog
    }

    func request(method: String, params: JSONDictionary) async throws -> JSONDictionary {
        try await ensureConnected()
        let requestID = nextRequestID
        nextRequestID += 1
        return try await withCheckedThrowingContinuation { continuation in
            pendingRequests[requestID] = continuation
            Task {
                do {
                    try await sendPayload([
                        "jsonrpc": "2.0",
                        "id": requestID,
                        "method": method,
                        "params": params,
                    ])
                } catch {
                    failRequest(id: requestID, error: error)
                }
            }
        }
    }

    func close() async {
        failAll(with: NativeHostBridgeError.webSocketClosed("App server websocket closed."))
        receiveLoopTask?.cancel()
        socket?.cancel(with: .normalClosure, reason: nil)
        session?.invalidateAndCancel()
        socket = nil
        session = nil
        readyTask = nil
    }

    private func ensureConnected() async throws {
        if let readyTask {
            try await readyTask.value
            return
        }
        let task = Task<Void, Error> {
            try await connect()
        }
        readyTask = task
        do {
            try await task.value
        } catch {
            readyTask = nil
            throw error
        }
    }

    private func connect() async throws {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 15
        configuration.timeoutIntervalForResource = 15
        let session = URLSession(configuration: configuration)
        var request = URLRequest(url: webSocketURL)
        headers.forEach { request.setValue($1, forHTTPHeaderField: $0) }
        let socket = session.webSocketTask(with: request)

        self.session = session
        self.socket = socket
        socket.resume()
        receiveLoopTask = Task { [weak self] in
            guard let self else { return }
            await self.receiveLoop(using: socket)
        }

        _ = try await sendRequestDuringConnection(
            id: 1,
            method: "initialize",
            params: [
                "clientInfo": [
                    "name": "codex_mobile_host",
                    "title": "Codex Mobile Host",
                    "version": "0.1.0",
                ],
                "capabilities": [
                    "experimentalApi": true,
                ],
            ]
        )

        try await sendPayload([
            "jsonrpc": "2.0",
            "method": "initialized",
            "params": [:],
        ])
    }

    private func sendRequestDuringConnection(id: Int, method: String, params: JSONDictionary) async throws -> JSONDictionary {
        try await withCheckedThrowingContinuation { continuation in
            pendingRequests[id] = continuation
            Task {
                do {
                    try await sendPayload([
                        "jsonrpc": "2.0",
                        "id": id,
                        "method": method,
                        "params": params,
                    ])
                } catch {
                    failRequest(id: id, error: error)
                }
            }
        }
    }

    private func sendPayload(_ payload: JSONDictionary) async throws {
        guard let socket else {
            throw NativeHostBridgeError.webSocketClosed("App server websocket is not connected.")
        }
        try await socket.send(.string(jsonEncodedString(payload)))
    }

    private func receiveLoop(using socket: URLSessionWebSocketTask) async {
        do {
            while !Task.isCancelled {
                let message = try await socket.receive()
                let text: String
                switch message {
                case let .string(value):
                    text = value
                case let .data(data):
                    text = String(data: data, encoding: .utf8) ?? ""
                @unknown default:
                    continue
                }
                await handleMessage(text)
            }
        } catch {
            onLog("app server websocket failure", error)
            failAll(with: error)
            readyTask = nil
            self.socket = nil
        }
    }

    private func handleMessage(_ text: String) async {
        guard let payload = parseJSONValue(from: text) as? JSONDictionary else {
            return
        }
        if let id = payload["id"] as? Int {
            handleResponse(id: id, payload: payload)
            return
        }

        guard let method = payload["method"] as? String,
              !method.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return
        }
        let params = payload["params"] as? JSONDictionary ?? [:]
        await onNotification(method, deepCopyJSONObject(params))
    }

    private func handleResponse(id: Int, payload: JSONDictionary) {
        guard let continuation = pendingRequests.removeValue(forKey: id) else {
            return
        }
        if let errorObject = payload["error"] as? JSONDictionary {
            let trimmedMessage = (errorObject["message"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
            let message = trimmedMessage?.isEmpty == false ? trimmedMessage! : "App server request failed."
            continuation.resume(throwing: NativeHostBridgeError.requestFailed(message))
            return
        }
        let result = payload["result"] as? JSONDictionary ?? [:]
        continuation.resume(returning: deepCopyJSONObject(result))
    }

    private func failRequest(id: Int, error: Error) {
        guard let continuation = pendingRequests.removeValue(forKey: id) else {
            return
        }
        continuation.resume(throwing: error)
    }

    private func failAll(with error: Error) {
        let requests = pendingRequests
        pendingRequests.removeAll()
        requests.values.forEach { continuation in
            continuation.resume(throwing: error)
        }
    }
}
