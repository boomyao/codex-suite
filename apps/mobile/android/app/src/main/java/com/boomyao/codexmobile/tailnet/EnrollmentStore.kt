package com.boomyao.codexmobile.tailnet

import android.content.Context

class EnrollmentStore(context: Context) {
    private val preferences = context.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)

    fun saveEnrollment(payload: TailnetEnrollmentPayload) {
        preferences.edit()
            .putString(KEY_RAW_ENROLLMENT, payload.rawPayload)
            .putString(KEY_BRIDGE_ID, payload.bridgeId)
            .putString(KEY_BRIDGE_NAME, payload.bridgeName)
            .putString(KEY_BRIDGE_SERVER_ENDPOINT, payload.bridgeServerEndpoint)
            .putString(KEY_CONTROL_URL, payload.controlUrl)
            .putString(KEY_AUTH_KEY, payload.authKey)
            .putString(KEY_HOSTNAME, payload.hostname)
            .putString(KEY_LOGIN_MODE, payload.loginMode)
            .apply()
    }

    fun writeStatus(snapshot: TailnetStatusSnapshot) {
        preferences.edit()
            .putString(KEY_STATUS_STATE, snapshot.state)
            .putString(KEY_STATUS_MODE, snapshot.mode)
            .putString(KEY_STATUS_MESSAGE, snapshot.message)
            .putString(KEY_STATUS_BRIDGE_NAME, snapshot.bridgeName)
            .putString(KEY_STATUS_BRIDGE_SERVER_ENDPOINT, snapshot.bridgeServerEndpoint)
            .putString(KEY_STATUS_LOCAL_PROXY_URL, snapshot.localProxyUrl)
            .putString(KEY_STATUS_RAW_ENROLLMENT_TYPE, snapshot.rawEnrollmentType)
            .putString(KEY_STATUS_AUTH_JSON, snapshot.auth?.toJson()?.toString())
            .putLong(KEY_STATUS_UPDATED_AT_MS, snapshot.updatedAtMs)
            .apply()
    }

    fun readStatus(): TailnetStatusSnapshot {
        val state = preferences.getString(KEY_STATUS_STATE, null) ?: "idle"
        val mode = preferences.getString(KEY_STATUS_MODE, null) ?: "native-shell"
        val message = preferences.getString(KEY_STATUS_MESSAGE, null)
            ?: "Embedded Android tailnet runtime has not started yet."
        return TailnetStatusSnapshot(
            state = state,
            mode = mode,
            message = message,
            bridgeName = preferences.getString(KEY_STATUS_BRIDGE_NAME, null),
            bridgeServerEndpoint = preferences.getString(KEY_STATUS_BRIDGE_SERVER_ENDPOINT, null),
            localProxyUrl = preferences.getString(KEY_STATUS_LOCAL_PROXY_URL, null),
            rawEnrollmentType = preferences.getString(KEY_STATUS_RAW_ENROLLMENT_TYPE, null),
            auth = preferences.getString(KEY_STATUS_AUTH_JSON, null)?.let { raw ->
                runCatching { parseTailnetAuthStatus(org.json.JSONObject(raw)) }.getOrNull()
            },
            updatedAtMs = preferences.getLong(KEY_STATUS_UPDATED_AT_MS, System.currentTimeMillis()),
        )
    }

    fun readEnrollment(): TailnetEnrollmentPayload? {
        val rawPayload = preferences.getString(KEY_RAW_ENROLLMENT, null)?.trim().orEmpty()
        if (rawPayload.isEmpty()) {
            return null
        }
        return runCatching { parseTailnetEnrollmentPayload(rawPayload) }.getOrNull()
    }

    fun clear() {
        preferences.edit().clear().apply()
    }

    companion object {
        private const val PREFERENCES_NAME = "codex_tailnet"
        private const val KEY_RAW_ENROLLMENT = "raw_enrollment"
        private const val KEY_BRIDGE_ID = "bridge_id"
        private const val KEY_BRIDGE_NAME = "bridge_name"
        private const val KEY_BRIDGE_SERVER_ENDPOINT = "bridge_server_endpoint"
        private const val KEY_CONTROL_URL = "control_url"
        private const val KEY_AUTH_KEY = "auth_key"
        private const val KEY_HOSTNAME = "hostname"
        private const val KEY_LOGIN_MODE = "login_mode"
        private const val KEY_STATUS_STATE = "status_state"
        private const val KEY_STATUS_MODE = "status_mode"
        private const val KEY_STATUS_MESSAGE = "status_message"
        private const val KEY_STATUS_BRIDGE_NAME = "status_bridge_name"
        private const val KEY_STATUS_BRIDGE_SERVER_ENDPOINT = "status_bridge_server_endpoint"
        private const val KEY_STATUS_LOCAL_PROXY_URL = "status_local_proxy_url"
        private const val KEY_STATUS_RAW_ENROLLMENT_TYPE = "status_raw_enrollment_type"
        private const val KEY_STATUS_AUTH_JSON = "status_auth_json"
        private const val KEY_STATUS_UPDATED_AT_MS = "status_updated_at_ms"
    }
}
