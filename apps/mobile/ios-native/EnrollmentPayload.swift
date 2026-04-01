import Foundation

enum EnrollmentPayload: Equatable {
    case bridge(bridgeID: String?, name: String, serverEndpoint: String, pairingCode: String?)
}

private let enrollmentWrapperKeys = [
    "payload",
    "mobileEnrollmentPayload",
    "enrollmentPayload",
    "data",
    "result",
]

private func unwrapEnrollmentPayloadValue(_ candidate: Any?) -> JSONDictionary? {
    if let object = candidate as? JSONDictionary {
        return object
    }
    guard let text = candidate as? String,
          let data = text.data(using: .utf8),
          let object = try? jsonObject(from: data) else {
        return nil
    }
    return object
}

private func normalizedEnrollmentPayloadObject(_ rawPayload: String) throws -> JSONDictionary {
    guard let data = rawPayload.data(using: .utf8) else {
        throw NativeHostBridgeError.invalidPayload("Enrollment payload is not valid JSON.")
    }
    var current = try jsonObject(from: data)
    for _ in 0..<4 {
        let payloadType = (current["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !payloadType.isEmpty {
            return current
        }
        guard let next = enrollmentWrapperKeys.compactMap({ unwrapEnrollmentPayloadValue(current[$0]) }).first else {
            return current
        }
        current = next
    }
    return current
}

private func normalizedEnrollmentPayloadJSON(_ rawPayload: String) throws -> String {
    jsonEncodedString(try normalizedEnrollmentPayloadObject(rawPayload))
}

enum EnrollmentParser {
    static func parse(rawJSON: String) throws -> EnrollmentPayload {
        let normalizedRawJSON = try normalizedEnrollmentPayloadJSON(rawJSON)
        let object = try normalizedEnrollmentPayloadObject(normalizedRawJSON)
        let type = (object["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

        switch type {
        case "codex-mobile-bridge":
            return .bridge(
                bridgeID: (object["bridgeId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                name: (object["name"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Codex Bridge",
                serverEndpoint: BridgeAPI.normalizeEndpoint((object["serverEndpoint"] as? String) ?? ""),
                pairingCode: (object["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            )
        case "codex-mobile-enrollment":
            return .bridge(
                bridgeID: (object["bridgeId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                name: (object["bridgeName"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "Codex Bridge",
                serverEndpoint: BridgeAPI.normalizeEndpoint((object["bridgeServerEndpoint"] as? String) ?? ""),
                pairingCode: (object["pairingCode"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            )
        default:
            throw NativeHostBridgeError.invalidPayload(unsupportedPayloadMessage(for: object, type: type))
        }
    }

    private static func unsupportedPayloadMessage(for object: JSONDictionary, type: String) -> String {
        let typeDescription = type.isEmpty ? "missing" : type
        let keysDescription = object.keys.sorted().joined(separator: ",").isEmpty ? "<none>" : object.keys.sorted().joined(separator: ",")
        return "Unsupported enrollment payload type (\(typeDescription)). Top-level keys: \(keysDescription)."
    }
}
