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
    val authKey: String,
    val hostname: String?,
    val loginMode: String?,
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
    val root = try {
        JSONObject(rawPayload)
    } catch (error: Exception) {
        throw IllegalArgumentException("Enrollment payload is not valid JSON.", error)
    }

    val payloadType = root.optString("type").trim()
    val version = root.optInt("version", -1)
    if (payloadType != "codex-mobile-enrollment" || version != 1) {
        throw IllegalArgumentException("Unsupported enrollment payload type.")
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
    val authKey = tailnet.optString("authKey").trim()
    if (authKey.isEmpty()) {
        throw IllegalArgumentException("Enrollment payload is missing tailnet.authKey.")
    }

    return TailnetEnrollmentPayload(
        rawPayload = rawPayload,
        bridgeId = root.optString("bridgeId").trim().ifEmpty { null },
        bridgeName = bridgeName,
        bridgeServerEndpoint = bridgeServerEndpoint,
        pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
        controlUrl = controlUrl,
        authKey = authKey,
        hostname = tailnet.optString("hostname").trim().ifEmpty { null },
        loginMode = tailnet.optString("loginMode").trim().ifEmpty { null },
    )
}
