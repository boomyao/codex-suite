package com.boomyao.codexmobile.tailnet

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKeys
import java.security.MessageDigest

data class TailnetEnrollmentSecrets(
    val clientSecret: String,
)

class TailnetCredentialStore(context: Context) {
    private val appContext = context.applicationContext
    private val preferences by lazy {
        val masterKeyAlias = MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC)
        EncryptedSharedPreferences.create(
            PREFERENCES_NAME,
            masterKeyAlias,
            appContext,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    fun save(payload: TailnetEnrollmentPayload) {
        save(
            rawPayload = payload.rawPayload,
            clientSecret = payload.clientSecret,
        )
    }

    fun save(rawPayload: String) {
        val payload = runCatching { parseTailnetEnrollmentPayload(rawPayload) }.getOrNull() ?: return
        save(payload)
    }

    fun save(rawPayload: String, clientSecret: String?) {
        val storageKey = storageKey(rawPayload) ?: return
        preferences.edit()
            .remove(key(storageKey, KEY_LEGACY_INLINE_SECRET))
            .putString(key(storageKey, KEY_CLIENT_SECRET), clientSecret?.trim()?.ifEmpty { null })
            .apply()
    }

    fun read(rawPayload: String): TailnetEnrollmentSecrets? {
        val storageKey = storageKey(rawPayload) ?: return null
        val legacyInlineSecret =
            preferences.getString(key(storageKey, KEY_LEGACY_INLINE_SECRET), null)?.trim()?.ifEmpty { null }
        val clientSecret =
            preferences.getString(key(storageKey, KEY_CLIENT_SECRET), null)?.trim()?.ifEmpty { null }
        if (legacyInlineSecret != null) {
            preferences.edit().remove(key(storageKey, KEY_LEGACY_INLINE_SECRET)).apply()
        }
        if (clientSecret == null) {
            return null
        }
        return TailnetEnrollmentSecrets(
            clientSecret = clientSecret,
        )
    }

    fun remove(rawPayload: String) {
        val storageKey = storageKey(rawPayload) ?: return
        preferences.edit()
            .remove(key(storageKey, KEY_LEGACY_INLINE_SECRET))
            .remove(key(storageKey, KEY_CLIENT_SECRET))
            .apply()
    }

    fun clear() {
        preferences.edit().clear().apply()
    }

    private fun storageKey(rawPayload: String): String? {
        val lookupKey = tailnetCredentialLookupKeyForPayload(rawPayload) ?: return null
        return sha256Hex(lookupKey)
    }

    private fun sha256Hex(value: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val bytes = digest.digest(value.toByteArray(Charsets.UTF_8))
        return bytes.joinToString(separator = "") { byte -> "%02x".format(byte) }
    }

    private fun key(storageKey: String, suffix: String): String = "${storageKey}:${suffix}"

    companion object {
        private const val PREFERENCES_NAME = "codex_tailnet_secure"
        private const val KEY_LEGACY_INLINE_SECRET = "authKey"
        private const val KEY_CLIENT_SECRET = "clientSecret"
    }
}
