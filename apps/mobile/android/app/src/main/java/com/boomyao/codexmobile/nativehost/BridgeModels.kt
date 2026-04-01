package com.boomyao.codexmobile.nativehost

import org.json.JSONObject

data class BridgeProfile(
    val id: String,
    val bridgeId: String?,
    val name: String,
    val serverEndpoint: String,
    val authToken: String?,
    val lastUsedAtMillis: Long? = null,
)

data class ConnectionTargetResponse(
    val bridgeId: String?,
    val recommendedServerEndpoint: String,
    val authMode: String,
    val localAuthPage: String?,
)

data class PairingResponse(
    val accessToken: String,
    val approved: Boolean,
)

data class BridgeLoadTarget(
    val baseUrl: String,
    val usesLocalProxy: Boolean,
)

data class HttpProxyResponse(
    val body: Any?,
    val status: Int,
    val headers: JSONObject,
)

data class BridgeBootstrapState(
    val persistedAtomState: JSONObject?,
    val workspaceRootOptions: List<String>,
    val activeWorkspaceRoots: List<String>,
    val workspaceRootLabels: Map<String, String>,
    val pinnedThreadIds: List<String>,
    val globalState: Map<String, Any?>,
)

sealed class EnrollmentPayload {
    data class Bridge(
        val bridgeId: String?,
        val name: String,
        val serverEndpoint: String,
        val pairingCode: String?,
    ) : EnrollmentPayload()
}
