package com.boomyao.codexmobile.nativehost

import org.json.JSONObject

data class BridgeProfile(
    val id: String,
    val name: String,
    val serverEndpoint: String,
    val authToken: String?,
    val tailnetEnrollmentPayload: String? = null,
    val lastUsedAtMillis: Long? = null,
)

data class ConnectionTargetResponse(
    val recommendedServerEndpoint: String,
    val authMode: String,
    val localAuthPage: String?,
)

data class PairingResponse(
    val accessToken: String,
    val approved: Boolean,
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
        val name: String,
        val serverEndpoint: String,
        val pairingCode: String?,
        val rawJson: String,
    ) : EnrollmentPayload()

    data class Tailnet(
        val bridgeName: String,
        val bridgeServerEndpoint: String,
        val pairingCode: String?,
        val rawJson: String,
    ) : EnrollmentPayload()
}
