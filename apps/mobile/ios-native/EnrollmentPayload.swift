import Foundation

enum EnrollmentPayload {
    case bridge(name: String, serverEndpoint: String, pairingCode: String?, rawJSON: String)
    case tailnet(bridgeName: String, bridgeServerEndpoint: String, pairingCode: String?, rawJSON: String)
}

enum EnrollmentParser {
    static func parse(rawJSON: String) throws -> EnrollmentPayload {
        let data = Data(rawJSON.utf8)
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        let normalizedObject = try normalizePayloadObject(object)
        let type = (normalizedObject["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let normalizedRawJSON = try normalizedJSONString(from: normalizedObject)

        switch type {
        case "codex-mobile-bridge":
            return .bridge(
                name: ((normalizedObject["name"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "Codex Bridge",
                serverEndpoint: ((normalizedObject["serverEndpoint"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                pairingCode: ((normalizedObject["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 },
                rawJSON: normalizedRawJSON
            )
        case "codex-mobile-enrollment":
            return .tailnet(
                bridgeName: ((normalizedObject["bridgeName"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "Codex Bridge",
                bridgeServerEndpoint: ((normalizedObject["bridgeServerEndpoint"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                pairingCode: ((normalizedObject["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 },
                rawJSON: normalizedRawJSON
            )
        default:
            throw NSError(domain: "CodexMobileHost", code: 1, userInfo: [
                NSLocalizedDescriptionKey: unsupportedPayloadMessage(for: normalizedObject)
            ])
        }
    }

    private static func normalizePayloadObject(_ object: [String: Any]) throws -> [String: Any] {
        let wrapperKeys = ["payload", "mobileEnrollmentPayload", "enrollmentPayload", "data", "result"]
        var current = object
        for _ in 0..<4 {
            let type = (current["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if !type.isEmpty {
                return current
            }
            var next: [String: Any]?
            for key in wrapperKeys {
                if let nested = current[key] as? [String: Any] {
                    next = nested
                    break
                }
                if let nestedJSON = current[key] as? String {
                    let nestedData = Data(nestedJSON.utf8)
                    if let nested = try? JSONSerialization.jsonObject(with: nestedData) as? [String: Any] {
                        next = nested
                        break
                    }
                }
            }
            guard let next else {
                return current
            }
            current = next
        }
        return current
    }

    private static func normalizedJSONString(from object: [String: Any]) throws -> String {
        let data = try JSONSerialization.data(withJSONObject: object)
        guard let json = String(data: data, encoding: .utf8) else {
            throw NSError(domain: "CodexMobileHost", code: 1, userInfo: [
                NSLocalizedDescriptionKey: "Enrollment payload is not valid JSON."
            ])
        }
        return json
    }

    private static func unsupportedPayloadMessage(for object: [String: Any]) -> String {
        let type = (object["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let typeDescription = type.isEmpty ? "missing" : type
        let keysDescription = object.keys.sorted().joined(separator: ",")
        return "Unsupported enrollment payload type (\(typeDescription)). Top-level keys: \(keysDescription.isEmpty ? "<none>" : keysDescription)."
    }
}
