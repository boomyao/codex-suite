import Foundation

struct ConnectionTargetResponse {
    var recommendedServerEndpoint: String
    var authMode: String
    var localAuthPage: String?
}

struct PairingResponse {
    var accessToken: String
    var approved: Bool
}

enum BridgeAPI {
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

    static func buildRemoteShellURL(_ endpoint: String) -> URL? {
        URL(string: deriveServerHTTPBaseURL(endpoint) + "/ui/index.html")
    }
}
