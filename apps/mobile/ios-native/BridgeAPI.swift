import Foundation
struct BridgeReadyProbe: Equatable {
    let ready: Bool
    let errorMessage: String?
}

enum BridgeAPI {
    private static let defaultTimeout: TimeInterval = 10

    static func normalizeEndpoint(_ value: String) -> String {
        var text = value.trimmingCharacters(in: .whitespacesAndNewlines)
        while text.hasSuffix("/") {
            text.removeLast()
        }
        return text
    }

    static func deriveServerHTTPBaseURL(_ endpoint: String) -> String {
        let normalized = normalizeEndpoint(endpoint)
        if normalized.hasPrefix("ws://") {
            return "http://" + normalized.dropFirst(5)
        }
        if normalized.hasPrefix("wss://") {
            return "https://" + normalized.dropFirst(6)
        }
        if normalized.hasPrefix("http://") || normalized.hasPrefix("https://") {
            return normalized
        }
        return "http://" + normalized
    }

    static func buildRemoteShellURL(baseURL: String, theme: String) -> URL? {
        var components = URLComponents(string: normalizeEndpoint(baseURL) + "/ui/index.html")
        components?.queryItems = [URLQueryItem(name: "codexTheme", value: theme)]
        return components?.url
    }

    static func fetchConnectionTarget(baseURL: String, authToken: String?) async throws -> ConnectionTargetResponse {
        let payload = try await requestJSON(
            url: normalizeEndpoint(baseURL) + "/codex-mobile/connect",
            authToken: authToken
        )
        guard let object = payload as? JSONDictionary else {
            throw NativeHostBridgeError.invalidResponse("Bridge connect response was not a JSON object.")
        }
        let connection = object["connection"] as? JSONDictionary
        let auth = object["auth"] as? JSONDictionary
        let recommendedServerEndpoint =
            (connection?["recommendedServerEndpoint"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty ?? normalizeEndpoint(baseURL)
        let localAuthPage = (object["localAuthPage"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty
        return ConnectionTargetResponse(
            bridgeID: (connection?["bridgeId"] as? String)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .nilIfEmpty,
            recommendedServerEndpoint: recommendedServerEndpoint,
            authMode: (auth?["mode"] as? String) == "device-token" ? "device-token" : "none",
            localAuthPage: localAuthPage
        )
    }

    static func completeDevicePairing(baseURL: String, pairingCode: String, authToken: String?) async throws -> PairingResponse {
        let requestBody: JSONDictionary = [
            "code": pairingCode.trimmingCharacters(in: .whitespacesAndNewlines),
            "deviceName": "Codex Mobile (iOS Native Host)",
        ]
        let payload = try await requestJSON(
            url: normalizeEndpoint(baseURL) + "/auth/pair/complete",
            method: "POST",
            authToken: authToken,
            requestBody: requestBody
        )
        guard let object = payload as? JSONDictionary else {
            throw NativeHostBridgeError.invalidResponse("Pairing response was not a JSON object.")
        }
        let accessToken = (object["accessToken"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !accessToken.isEmpty else {
            throw NativeHostBridgeError.invalidResponse("Pairing response did not include an access token.")
        }
        return PairingResponse(accessToken: accessToken, approved: object["approved"] as? Bool ?? false)
    }

    static func performDirectRPC(baseURL: String, method: String, params: JSONDictionary?, authToken: String?) async throws -> JSONDictionary {
        let requestBody: JSONDictionary = [
            "method": method,
            "params": params ?? [:],
        ]
        let payload = try await requestJSON(
            url: normalizeEndpoint(baseURL) + "/codex-mobile/rpc",
            method: "POST",
            authToken: authToken,
            requestBody: requestBody
        )
        guard let object = payload as? JSONDictionary else {
            throw NativeHostBridgeError.invalidResponse("Direct RPC response was not a JSON object.")
        }
        guard object["ok"] as? Bool == true else {
            let error = (object["error"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            throw NativeHostBridgeError.requestFailed(error?.isEmpty == false ? error! : "Direct RPC failed.")
        }
        return object["result"] as? JSONDictionary ?? [:]
    }

    static func fetchGlobalStateSnapshot(baseURL: String, authToken: String?) async throws -> JSONDictionary {
        var lastError: Error?
        for attempt in 0..<6 {
            do {
                return try await performDirectRPC(
                    baseURL: baseURL,
                    method: "get-global-state-snapshot",
                    params: [:],
                    authToken: authToken
                )
            } catch {
                lastError = error
                if attempt == 5 {
                    throw error
                }
                try await Task.sleep(nanoseconds: 250_000_000)
            }
        }
        throw lastError ?? NativeHostBridgeError.requestFailed("Failed to fetch bridge global state snapshot.")
    }

    static func probeBridgeReady(baseURL: String, authToken: String?) async -> BridgeReadyProbe {
        do {
            _ = try await requestJSON(
                url: normalizeEndpoint(baseURL) + "/readyz",
                authToken: authToken
            )
            return BridgeReadyProbe(ready: true, errorMessage: nil)
        } catch {
            return BridgeReadyProbe(
                ready: false,
                errorMessage: (error as NSError).localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            )
        }
    }

    static func proxyHTTPFetch(
        url: URL,
        method: String?,
        requestBody: String?,
        requestHeaders: JSONDictionary?,
        authToken: String?
    ) async throws -> HttpProxyResponse {
        var request = URLRequest(url: url, timeoutInterval: defaultTimeout)
        request.httpMethod = method?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased().nilIfEmpty ?? "GET"
        requestHeaders?.forEach { key, value in
            if let text = value as? String, !text.isEmpty {
                request.setValue(text, forHTTPHeaderField: key)
            }
        }
        if request.value(forHTTPHeaderField: "Authorization").nilIfEmpty == nil,
           let authToken, !authToken.isEmpty {
            request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        }
        if let requestBody, !requestBody.isEmpty {
            request.httpBody = Data(requestBody.utf8)
            if request.value(forHTTPHeaderField: "Content-Type").nilIfEmpty == nil {
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            }
        }
        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw NativeHostBridgeError.invalidResponse("Proxy response did not include HTTP metadata.")
        }
        let headers = httpResponse.allHeaderFields.reduce(into: JSONDictionary()) { partialResult, entry in
            if let key = entry.key as? String {
                partialResult[key] = "\(entry.value)"
            }
        }
        let text = String(data: data, encoding: .utf8) ?? ""
        return HttpProxyResponse(
            body: parseJSONValue(from: text),
            status: httpResponse.statusCode,
            headers: headers
        )
    }

    private static func requestJSON(
        url: String,
        method: String = "GET",
        authToken: String? = nil,
        requestBody: JSONDictionary? = nil
    ) async throws -> Any {
        guard let requestURL = URL(string: url) else {
            throw NativeHostBridgeError.invalidURL("Invalid bridge URL: \(url)")
        }
        var request = URLRequest(url: requestURL, timeoutInterval: defaultTimeout)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let authToken, !authToken.isEmpty {
            request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        }
        if let requestBody {
            request.httpBody = try JSONSerialization.data(withJSONObject: requestBody, options: [])
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = defaultTimeout
        configuration.timeoutIntervalForResource = defaultTimeout
        let session = URLSession(configuration: configuration)
        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw NativeHostBridgeError.invalidResponse("Bridge response did not include HTTP metadata.")
        }
        let responseText = String(data: data, encoding: .utf8) ?? ""
        if !(200...299).contains(httpResponse.statusCode) {
            let message: String
            if let errorObject = parseJSONValue(from: responseText) as? JSONDictionary,
               let errorText = (errorObject["error"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
               !errorText.isEmpty {
                message = errorText
            } else if !responseText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                message = responseText.trimmingCharacters(in: .whitespacesAndNewlines)
            } else {
                message = "Request failed with HTTP \(httpResponse.statusCode)."
            }
            throw NativeHostBridgeError.requestFailed(message)
        }

        if responseText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return [:]
        }
        guard let object = parseJSONValue(from: responseText) else {
            throw NativeHostBridgeError.invalidResponse("Bridge response was not valid JSON.")
        }
        return object
    }
}
