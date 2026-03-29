import Foundation

enum EnrollmentPayload {
    case bridge(name: String, serverEndpoint: String, pairingCode: String?, rawJSON: String)
    case tailnet(bridgeName: String, bridgeServerEndpoint: String, pairingCode: String?, rawJSON: String)
}

enum EnrollmentParser {
    static func parse(rawJSON: String) throws -> EnrollmentPayload {
        let data = Data(rawJSON.utf8)
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        let type = (object["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

        switch type {
        case "codex-mobile-bridge":
            return .bridge(
                name: ((object["name"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "Codex Bridge",
                serverEndpoint: ((object["serverEndpoint"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                pairingCode: ((object["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 },
                rawJSON: rawJSON
            )
        case "codex-mobile-enrollment":
            return .tailnet(
                bridgeName: ((object["bridgeName"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "Codex Bridge",
                bridgeServerEndpoint: ((object["bridgeServerEndpoint"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                pairingCode: ((object["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 },
                rawJSON: rawJSON
            )
        default:
            throw NSError(domain: "CodexMobileHost", code: 1, userInfo: [
                NSLocalizedDescriptionKey: "Unsupported enrollment payload type."
            ])
        }
    }
}
