package com.boomyao.codexmobile.tailnet

import org.json.JSONObject

data class TailnetAuthStatus(
    val backend: String?,
    val backendState: String?,
    val loggedIn: Boolean,
    val needsLogin: Boolean,
    val needsMachineAuth: Boolean,
    val authUrl: String?,
    val tailnet: String?,
    val selfDnsName: String?,
    val magicDnsSuffix: String?,
    val magicDnsEnabled: Boolean,
    val tailscaleIps: List<String>,
    val health: List<String>,
)

data class TailnetEnrollmentPayload(
    val rawPayload: String,
    val bridgeId: String?,
    val bridgeName: String,
    val bridgeServerEndpoint: String,
    val pairingCode: String?,
    val controlUrl: String,
    val hostname: String?,
    val loginMode: String?,
    val oauthClientId: String,
    val oauthTailnet: String,
    val oauthTags: List<String>,
    val clientSecret: String,
)

data class TailnetStatusSnapshot(
    val state: String,
    val mode: String,
    val message: String,
    val bridgeName: String?,
    val bridgeServerEndpoint: String?,
    val localProxyUrl: String?,
    val rawEnrollmentType: String?,
    val auth: TailnetAuthStatus?,
    val updatedAtMs: Long = System.currentTimeMillis(),
)

private const val LEGACY_INLINE_SECRET_JSON_KEY = "authKey"
private const val CLIENT_SECRET_JSON_KEY = "clientSecret"

private val ENROLLMENT_WRAPPER_KEYS = listOf(
    "payload",
    "mobileEnrollmentPayload",
    "enrollmentPayload",
    "data",
    "result",
)

private fun JSONObject.keyNames(): List<String> {
    val iterator = keys()
    val values = mutableListOf<String>()
    while (iterator.hasNext()) {
        values += iterator.next()
    }
    return values.sorted()
}

private fun unwrapEnrollmentPayloadValue(candidate: Any?): JSONObject? {
    return when (candidate) {
        is JSONObject -> candidate
        is String -> runCatching { JSONObject(candidate) }.getOrNull()
        else -> null
    }
}

private fun unwrapEnrollmentPayloadRoot(root: JSONObject): JSONObject {
    var current = root
    repeat(4) {
        val payloadType = current.optString("type").trim()
        if (payloadType.isNotEmpty()) {
            return current
        }
        val next =
            ENROLLMENT_WRAPPER_KEYS.firstNotNullOfOrNull { key ->
                unwrapEnrollmentPayloadValue(current.opt(key))
            }
        if (next == null) {
            return current
        }
        current = next
    }
    return current
}

private fun unsupportedEnrollmentPayloadTypeError(root: JSONObject): IllegalArgumentException {
    val payloadType = root.optString("type").trim()
    val typeDescription =
        if (payloadType.isEmpty()) {
            "missing"
        } else {
            payloadType
        }
    val keysDescription = root.keyNames().joinToString(",").ifEmpty { "<none>" }
    return IllegalArgumentException(
        "Unsupported enrollment payload type ($typeDescription). Top-level keys: $keysDescription.",
    )
}

private fun normalizedEnrollmentPayloadRoot(rawPayload: String): JSONObject {
    val root = try {
        JSONObject(rawPayload)
    } catch (error: Exception) {
        throw IllegalArgumentException("Enrollment payload is not valid JSON.", error)
    }
    return unwrapEnrollmentPayloadRoot(root)
}

fun normalizedEnrollmentPayloadJson(rawPayload: String): String {
    return normalizedEnrollmentPayloadRoot(rawPayload).toString()
}

fun tailnetCredentialLookupKeyForPayload(rawPayload: String): String? {
    val root = runCatching { normalizedEnrollmentPayloadRoot(rawPayload) }.getOrNull() ?: return null
    if (root.optString("type").trim() != "codex-mobile-enrollment") {
        return null
    }
    val bridgeId = root.optString("bridgeId").trim()
    if (bridgeId.isNotEmpty()) {
        return "bridgeId:$bridgeId"
    }
    val bridgeServerEndpoint = root.optString("bridgeServerEndpoint").trim()
    if (bridgeServerEndpoint.isNotEmpty()) {
        return "bridgeEndpoint:$bridgeServerEndpoint"
    }
    return null
}

fun sanitizeTailnetEnrollmentPayloadForStorage(rawPayload: String): String {
    val root = runCatching { normalizedEnrollmentPayloadRoot(rawPayload) }.getOrNull() ?: return rawPayload
    if (root.optString("type").trim() != "codex-mobile-enrollment") {
        return rawPayload
    }
    val tailnet = root.optJSONObject("tailnet") ?: return rawPayload
    tailnet.remove(LEGACY_INLINE_SECRET_JSON_KEY)
    tailnet.remove(CLIENT_SECRET_JSON_KEY)
    return root.toString()
}

fun tailnetEnrollmentContainsInlineSecrets(rawPayload: String): Boolean {
    val root = runCatching { normalizedEnrollmentPayloadRoot(rawPayload) }.getOrNull() ?: return false
    if (root.optString("type").trim() != "codex-mobile-enrollment") {
        return false
    }
    val tailnet = root.optJSONObject("tailnet") ?: return false
    return tailnet.optString(LEGACY_INLINE_SECRET_JSON_KEY).trim().isNotEmpty() ||
        tailnet.optString(CLIENT_SECRET_JSON_KEY).trim().isNotEmpty()
}

fun restoreTailnetEnrollmentPayload(
    rawPayload: String,
    secrets: TailnetEnrollmentSecrets?,
): String {
    if (secrets == null) {
        return rawPayload
    }
    val root = runCatching { normalizedEnrollmentPayloadRoot(rawPayload) }.getOrNull() ?: return rawPayload
    if (root.optString("type").trim() != "codex-mobile-enrollment") {
        return rawPayload
    }
    val tailnet = root.optJSONObject("tailnet") ?: return rawPayload
    tailnet.put(CLIENT_SECRET_JSON_KEY, secrets.clientSecret)
    return root.toString()
}

fun TailnetAuthStatus.toJson(): JSONObject {
    return JSONObject()
        .put("backend", backend)
        .put("backendState", backendState)
        .put("loggedIn", loggedIn)
        .put("needsLogin", needsLogin)
        .put("needsMachineAuth", needsMachineAuth)
        .put("authUrl", authUrl)
        .put("tailnet", tailnet)
        .put("selfDnsName", selfDnsName)
        .put("magicDnsSuffix", magicDnsSuffix)
        .put("magicDnsEnabled", magicDnsEnabled)
        .put("tailscaleIps", tailscaleIps)
        .put("health", health)
}

fun parseTailnetAuthStatus(root: JSONObject?): TailnetAuthStatus? {
    if (root == null) {
        return null
    }
    val tailscaleIps = mutableListOf<String>()
    val tailscaleIpsJson = root.optJSONArray("tailscaleIps")
    if (tailscaleIpsJson != null) {
        for (index in 0 until tailscaleIpsJson.length()) {
            val value = tailscaleIpsJson.optString(index).trim()
            if (value.isNotEmpty()) {
                tailscaleIps += value
            }
        }
    }
    val health = mutableListOf<String>()
    val healthJson = root.optJSONArray("health")
    if (healthJson != null) {
        for (index in 0 until healthJson.length()) {
            val value = healthJson.optString(index).trim()
            if (value.isNotEmpty()) {
                health += value
            }
        }
    }
    return TailnetAuthStatus(
        backend = root.optString("backend").trim().ifEmpty { null },
        backendState = root.optString("backendState").trim().ifEmpty { null },
        loggedIn = root.optBoolean("loggedIn", false),
        needsLogin = root.optBoolean("needsLogin", false),
        needsMachineAuth = root.optBoolean("needsMachineAuth", false),
        authUrl = root.optString("authUrl").trim().ifEmpty { null },
        tailnet = root.optString("tailnet").trim().ifEmpty { null },
        selfDnsName = root.optString("selfDnsName").trim().ifEmpty { null },
        magicDnsSuffix = root.optString("magicDnsSuffix").trim().ifEmpty { null },
        magicDnsEnabled = root.optBoolean("magicDnsEnabled", false),
        tailscaleIps = tailscaleIps,
        health = health,
    )
}

fun parseTailnetEnrollmentPayload(rawPayload: String): TailnetEnrollmentPayload {
    val normalizedRawPayload = normalizedEnrollmentPayloadJson(rawPayload)
    val root = JSONObject(normalizedRawPayload)

    val payloadType = root.optString("type").trim()
    if (payloadType != "codex-mobile-enrollment") {
        throw unsupportedEnrollmentPayloadTypeError(root)
    }

    val bridgeServerEndpoint = root.optString("bridgeServerEndpoint").trim()
    if (bridgeServerEndpoint.isEmpty()) {
        throw IllegalArgumentException("Enrollment payload is missing bridgeServerEndpoint.")
    }

    val bridgeName = root.optString("bridgeName").trim().ifEmpty {
        root.optString("name").trim().ifEmpty { "Codex Bridge" }
    }
    val tailnet = root.optJSONObject("tailnet")
        ?: throw IllegalArgumentException("Enrollment payload is missing tailnet settings.")
    val controlUrl = tailnet.optString("controlUrl").trim()
    val clientSecret = tailnet.optString(CLIENT_SECRET_JSON_KEY).trim().ifEmpty { null }
    if (clientSecret == null) {
        throw IllegalArgumentException("Enrollment payload is missing tailnet.clientSecret.")
    }
    val oauthClientId = tailnet.optString("oauthClientId").trim().ifEmpty { null }
        ?: throw IllegalArgumentException("Enrollment payload is missing tailnet.oauthClientId.")
    val oauthTailnet = tailnet.optString("oauthTailnet").trim().ifEmpty { null }
        ?: throw IllegalArgumentException("Enrollment payload is missing tailnet.oauthTailnet.")
    val oauthTags = mutableListOf<String>()
    val oauthTagsJson = tailnet.optJSONArray("oauthTags")
    if (oauthTagsJson != null) {
        for (index in 0 until oauthTagsJson.length()) {
            val value = oauthTagsJson.optString(index).trim()
            if (value.isNotEmpty()) {
                oauthTags += value
            }
        }
    }
    if (oauthTags.isEmpty()) {
        throw IllegalArgumentException("Enrollment payload must include tailnet.oauthTags.")
    }

    return TailnetEnrollmentPayload(
        rawPayload = normalizedRawPayload,
        bridgeId = root.optString("bridgeId").trim().ifEmpty { null },
        bridgeName = bridgeName,
        bridgeServerEndpoint = bridgeServerEndpoint,
        pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
        controlUrl = controlUrl,
        hostname = tailnet.optString("hostname").trim().ifEmpty { null },
        loginMode = tailnet.optString("loginMode").trim().ifEmpty { null },
        oauthClientId = oauthClientId,
        oauthTailnet = oauthTailnet,
        oauthTags = oauthTags,
        clientSecret = clientSecret,
    )
}
