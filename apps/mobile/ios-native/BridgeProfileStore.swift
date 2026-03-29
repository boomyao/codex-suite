import Foundation

struct BridgeProfile: Codable, Equatable {
    var name: String
    var serverEndpoint: String
    var authToken: String?
}

final class BridgeProfileStore {
    private let defaults = UserDefaults.standard
    private let key = "codex.nativeHost.bridgeProfile"

    func read() -> BridgeProfile? {
        guard let data = defaults.data(forKey: key) else { return nil }
        return try? JSONDecoder().decode(BridgeProfile.self, from: data)
    }

    func write(_ profile: BridgeProfile) {
        if let data = try? JSONEncoder().encode(profile) {
            defaults.set(data, forKey: key)
        }
    }

    func clear() {
        defaults.removeObject(forKey: key)
    }
}
