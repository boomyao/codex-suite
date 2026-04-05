package com.boomyao.codexmobile.nativehost

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject
import kotlin.math.abs

class BridgeProfileStore(context: Context) {
    private val preferences =
        context.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)

    fun list(): List<BridgeProfile> {
        val rawProfiles = preferences.getString(KEY_PROFILES_JSON, null)?.trim().orEmpty()
        if (rawProfiles.isNotEmpty()) {
            return persistSanitizedProfilesIfNeeded(parseProfiles(rawProfiles))
        }
        return migrateLegacyProfile()?.let(::listOf).orEmpty()
    }

    fun readActive(): BridgeProfile? {
        val profiles = list()
        if (profiles.isEmpty()) {
            return null
        }
        val activeProfileId = preferences.getString(KEY_ACTIVE_PROFILE_ID, null)?.trim().orEmpty()
        return profiles.firstOrNull { it.id == activeProfileId } ?: profiles.firstOrNull()
    }

    fun write(profile: BridgeProfile) {
        val now = System.currentTimeMillis()
        val profileToSave = profile.copy(lastUsedAtMillis = profile.lastUsedAtMillis ?: now)
        val profiles = list().toMutableList()
        profiles.removeAll { it.id == profileToSave.id }
        profiles.add(0, profileToSave)
        persistProfiles(profiles, profileToSave.id)
    }

    fun setActive(profileId: String) {
        val profiles = list().toMutableList()
        val activeProfile = profiles.firstOrNull { it.id == profileId }?.copy(lastUsedAtMillis = System.currentTimeMillis()) ?: return
        profiles.removeAll { it.id == activeProfile.id }
        profiles.add(0, activeProfile)
        persistProfiles(profiles, activeProfile.id)
    }

    fun remove(profileId: String): BridgeProfile? {
        val currentProfiles = list()
        val removedProfile = currentProfiles.firstOrNull { it.id == profileId }
        val remainingProfiles = currentProfiles.filterNot { it.id == profileId }
        val currentActiveId = preferences.getString(KEY_ACTIVE_PROFILE_ID, null)?.trim().orEmpty()
        val nextActiveId =
            when {
                remainingProfiles.isEmpty() -> null
                currentActiveId == profileId -> remainingProfiles.first().id
                remainingProfiles.any { it.id == currentActiveId } -> currentActiveId
                else -> remainingProfiles.first().id
            }
        persistProfiles(remainingProfiles, nextActiveId)
        return remainingProfiles.firstOrNull { it.id == nextActiveId }
    }

    fun clear() {
        preferences.edit().clear().apply()
    }

    fun createProfileId(name: String, endpoint: String, bridgeId: String? = null): String {
        val seed = "${bridgeId?.trim().orEmpty()}|${name.trim()}|${endpoint.trim()}|${System.currentTimeMillis()}"
        return "bridge_${System.currentTimeMillis()}_${abs(seed.hashCode())}"
    }

    private data class ParsedProfiles(
        val profiles: List<BridgeProfile>,
        val mutated: Boolean,
    )

    private fun parseProfiles(rawProfiles: String): ParsedProfiles {
        return runCatching {
            val root = JSONArray(rawProfiles)
            var mutated = false
            buildList {
                for (index in 0 until root.length()) {
                    val item = root.optJSONObject(index) ?: continue
                    val id = item.optString("id").trim()
                    val endpoint = item.optString("serverEndpoint").trim()
                    if (id.isEmpty() || endpoint.isEmpty()) {
                        mutated = true
                        continue
                    }
                    if (item.optString("tailnetEnrollmentPayload").trim().isNotEmpty()) {
                        mutated = true
                    }
                    add(
                        BridgeProfile(
                            id = id,
                            bridgeId = item.optString("bridgeId").trim().ifEmpty { null },
                            name = item.optString("name").trim().ifEmpty { endpoint },
                            serverEndpoint = endpoint,
                            authToken = item.optString("authToken").trim().ifEmpty { null },
                            lastUsedAtMillis = item.optLong("lastUsedAtMillis").takeIf { it > 0L },
                            libp2pPeerId = item.optString("libp2pPeerId").trim().ifEmpty { null },
                        ),
                    )
                }
            }.let { profiles ->
                ParsedProfiles(
                    profiles = profiles,
                    mutated = mutated,
                )
            }
        }.getOrDefault(ParsedProfiles(emptyList(), mutated = true))
    }

    private fun persistSanitizedProfilesIfNeeded(parsed: ParsedProfiles): List<BridgeProfile> {
        if (parsed.mutated) {
            val activeProfileId = preferences.getString(KEY_ACTIVE_PROFILE_ID, null)?.trim().orEmpty()
            val nextActiveId = parsed.profiles.firstOrNull { it.id == activeProfileId }?.id ?: parsed.profiles.firstOrNull()?.id
            persistProfiles(parsed.profiles, nextActiveId)
        }
        return parsed.profiles
    }

    private fun persistProfiles(profiles: List<BridgeProfile>, activeProfileId: String?) {
        val serializedProfiles =
            JSONArray().apply {
                profiles.forEach { profile ->
                    put(
                        JSONObject()
                            .put("id", profile.id)
                            .put("bridgeId", profile.bridgeId)
                            .put("name", profile.name)
                            .put("serverEndpoint", profile.serverEndpoint)
                            .put("authToken", profile.authToken)
                            .put("lastUsedAtMillis", profile.lastUsedAtMillis)
                            .put("libp2pPeerId", profile.libp2pPeerId)
                    )
                }
            }
        preferences.edit()
            .putString(KEY_PROFILES_JSON, serializedProfiles.toString())
            .putString(KEY_ACTIVE_PROFILE_ID, activeProfileId)
            .remove(KEY_NAME)
            .remove(KEY_SERVER_ENDPOINT)
            .remove(KEY_AUTH_TOKEN)
            .apply()
    }

    private fun migrateLegacyProfile(): BridgeProfile? {
        val endpoint = preferences.getString(KEY_SERVER_ENDPOINT, null)?.trim().orEmpty()
        if (endpoint.isEmpty()) {
            return null
        }
        val name = preferences.getString(KEY_NAME, null)?.trim().orEmpty().ifEmpty { endpoint }
        val authToken = preferences.getString(KEY_AUTH_TOKEN, null)?.trim()?.ifEmpty { null }
        val profile =
            BridgeProfile(
                id = createProfileId(name, endpoint),
                bridgeId = null,
                name = name,
                serverEndpoint = endpoint,
                authToken = authToken,
                lastUsedAtMillis = System.currentTimeMillis(),
            )
        persistProfiles(listOf(profile), profile.id)
        return profile
    }

    companion object {
        private const val PREFERENCES_NAME = "codex_native_host"
        private const val KEY_NAME = "name"
        private const val KEY_SERVER_ENDPOINT = "server_endpoint"
        private const val KEY_AUTH_TOKEN = "auth_token"
        private const val KEY_PROFILES_JSON = "profiles_json"
        private const val KEY_ACTIVE_PROFILE_ID = "active_profile_id"
    }
}
