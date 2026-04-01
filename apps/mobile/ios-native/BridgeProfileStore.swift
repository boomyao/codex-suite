import CryptoKit
import Foundation

final class BridgeProfileStore {
    private let defaults = UserDefaults.standard

    private let preferencesName = "codex.nativeHost.profiles"
    private let activeProfileKey = "codex.nativeHost.activeProfileID"
    private let legacyNameKey = "codex.nativeHost.legacy.name"
    private let legacyEndpointKey = "codex.nativeHost.legacy.serverEndpoint"
    private let legacyAuthTokenKey = "codex.nativeHost.legacy.authToken"

    func list() -> [BridgeProfile] {
        let rawProfiles = defaults.string(forKey: preferencesName)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !rawProfiles.isEmpty {
            return persistSanitizedProfilesIfNeeded(parseProfiles(rawProfiles))
        }
        return migrateLegacyProfile().map { [$0] } ?? []
    }

    func readActive() -> BridgeProfile? {
        let profiles = list()
        guard !profiles.isEmpty else {
            return nil
        }
        let activeProfileID = defaults.string(forKey: activeProfileKey)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return profiles.first(where: { $0.id == activeProfileID }) ?? profiles.first
    }

    func write(_ profile: BridgeProfile) {
        let now = Int64(Date().timeIntervalSince1970 * 1000)
        let profileToSave = BridgeProfile(
            id: profile.id,
            bridgeID: profile.bridgeID,
            name: profile.name,
            serverEndpoint: profile.serverEndpoint,
            authToken: profile.authToken,
            lastUsedAtMilliseconds: profile.lastUsedAtMilliseconds ?? now
        )

        var profiles = list()
        profiles.removeAll { $0.id == profileToSave.id }
        profiles.insert(profileToSave, at: 0)
        persistProfiles(profiles, activeProfileID: profileToSave.id)
    }

    func setActive(_ profileID: String) {
        var profiles = list()
        guard let activeProfile = profiles.first(where: { $0.id == profileID }) else {
            return
        }
        let refreshed = BridgeProfile(
            id: activeProfile.id,
            bridgeID: activeProfile.bridgeID,
            name: activeProfile.name,
            serverEndpoint: activeProfile.serverEndpoint,
            authToken: activeProfile.authToken,
            lastUsedAtMilliseconds: Int64(Date().timeIntervalSince1970 * 1000)
        )
        profiles.removeAll { $0.id == profileID }
        profiles.insert(refreshed, at: 0)
        persistProfiles(profiles, activeProfileID: refreshed.id)
    }

    @discardableResult
    func remove(_ profileID: String) -> BridgeProfile? {
        let currentProfiles = list()
        let remainingProfiles = currentProfiles.filter { $0.id != profileID }
        let currentActiveID = defaults.string(forKey: activeProfileKey)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let nextActiveID: String?
        if remainingProfiles.isEmpty {
            nextActiveID = nil
        } else if currentActiveID == profileID {
            nextActiveID = remainingProfiles.first?.id
        } else if remainingProfiles.contains(where: { $0.id == currentActiveID }) {
            nextActiveID = currentActiveID
        } else {
            nextActiveID = remainingProfiles.first?.id
        }
        persistProfiles(remainingProfiles, activeProfileID: nextActiveID)
        return remainingProfiles.first(where: { $0.id == nextActiveID })
    }

    func clear() {
        defaults.removeObject(forKey: preferencesName)
        defaults.removeObject(forKey: activeProfileKey)
    }

    func createProfileID(name: String, endpoint: String, bridgeID: String? = nil) -> String {
        let seed = "\(bridgeID?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "")|\(name.trimmingCharacters(in: .whitespacesAndNewlines))|\(endpoint.trimmingCharacters(in: .whitespacesAndNewlines))|\(Date().timeIntervalSince1970)"
        let digest = SHA256.hash(data: Data(seed.utf8))
        let suffix = digest.prefix(6).map { String(format: "%02x", $0) }.joined()
        return "bridge_\(Int(Date().timeIntervalSince1970))_\(suffix)"
    }

    private struct ParsedProfiles {
        let profiles: [BridgeProfile]
        let mutated: Bool
    }

    private func parseProfiles(_ rawProfiles: String) -> ParsedProfiles {
        guard let data = rawProfiles.data(using: .utf8),
              let root = try? JSONSerialization.jsonObject(with: data, options: []) as? [JSONDictionary] else {
            return ParsedProfiles(profiles: [], mutated: true)
        }

        var mutated = false
        let profiles = root.compactMap { item -> BridgeProfile? in
            let id = (item["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let endpoint = (item["serverEndpoint"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !id.isEmpty, !endpoint.isEmpty else {
                mutated = true
                return nil
            }
            if let legacyPayload = (item["tailnetEnrollmentPayload"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
               !legacyPayload.isEmpty {
                mutated = true
            }

            return BridgeProfile(
                id: id,
                bridgeID: (item["bridgeId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                name: (item["name"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? endpoint,
                serverEndpoint: endpoint,
                authToken: (item["authToken"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                lastUsedAtMilliseconds: (item["lastUsedAtMillis"] as? Int64) ?? (item["lastUsedAtMillis"] as? Int).map(Int64.init)
            )
        }
        return ParsedProfiles(profiles: profiles, mutated: mutated)
    }

    private func persistSanitizedProfilesIfNeeded(_ parsed: ParsedProfiles) -> [BridgeProfile] {
        guard parsed.mutated else {
            return parsed.profiles
        }
        let currentActiveID = defaults.string(forKey: activeProfileKey)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let nextActiveID = parsed.profiles.first(where: { $0.id == currentActiveID })?.id ?? parsed.profiles.first?.id
        persistProfiles(parsed.profiles, activeProfileID: nextActiveID)
        return parsed.profiles
    }

    private func persistProfiles(_ profiles: [BridgeProfile], activeProfileID: String?) {
        let serializedProfiles = profiles.map { profile in
            [
                "id": profile.id,
                "bridgeId": profile.bridgeID as Any,
                "name": profile.name,
                "serverEndpoint": profile.serverEndpoint,
                "authToken": profile.authToken as Any,
                "lastUsedAtMillis": profile.lastUsedAtMilliseconds as Any,
            ] as JSONDictionary
        }
        defaults.set(jsonEncodedString(serializedProfiles), forKey: preferencesName)
        defaults.set(activeProfileID, forKey: activeProfileKey)
        defaults.removeObject(forKey: legacyNameKey)
        defaults.removeObject(forKey: legacyEndpointKey)
        defaults.removeObject(forKey: legacyAuthTokenKey)
    }

    private func migrateLegacyProfile() -> BridgeProfile? {
        let endpoint = defaults.string(forKey: legacyEndpointKey)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !endpoint.isEmpty else {
            return nil
        }
        let name = defaults.string(forKey: legacyNameKey)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? endpoint
        let authToken = defaults.string(forKey: legacyAuthTokenKey)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
        let profile = BridgeProfile(
            id: createProfileID(name: name, endpoint: endpoint),
            bridgeID: nil,
            name: name,
            serverEndpoint: endpoint,
            authToken: authToken,
            lastUsedAtMilliseconds: Int64(Date().timeIntervalSince1970 * 1000)
        )
        persistProfiles([profile], activeProfileID: profile.id)
        return profile
    }
}
